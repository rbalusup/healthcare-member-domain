package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	appMember "github.com/healthcare/member-service/internal/application/member"
	"github.com/healthcare/member-service/internal/config"
	grpcHandler "github.com/healthcare/member-service/internal/handler/grpc"
	httpHandler "github.com/healthcare/member-service/internal/handler/http"
	"github.com/healthcare/member-service/internal/infrastructure/kafka"
	"github.com/healthcare/member-service/internal/infrastructure/metrics"
	"github.com/healthcare/member-service/internal/infrastructure/postgres"
	memberv1 "github.com/healthcare/member-service/gen/go/member/v1"
)

func main() {
	// ---- Signal context ----
	// Cancelled on SIGINT or SIGTERM; triggers graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ---- Logger ----
	logger, err := zap.NewProduction()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync() //nolint:errcheck

	// ---- Config ----
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	// ---- Metrics ----
	m := metrics.New()
	_ = m // used by application service via dependency injection in full impl

	// ---- Database ----
	db, err := postgres.Open(
		cfg.Database.DSN(),
		cfg.Database.MaxOpenConns,
		cfg.Database.MaxIdleConns,
		cfg.Database.ConnMaxLifetime,
	)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()
	logger.Info("database connected", zap.String("host", cfg.Database.Host))

	// ---- Kafka Producer ----
	producer, err := kafka.NewProducer(kafka.ProducerConfig{
		Brokers:         cfg.Kafka.Brokers,
		TransactionalID: cfg.Kafka.TransactionalID,
		Topics: kafka.Topics{
			MemberCreated:       cfg.Kafka.Topics.MemberCreated,
			MemberUpdated:       cfg.Kafka.Topics.MemberUpdated,
			MemberEnrolled:      cfg.Kafka.Topics.MemberEnrolled,
			MemberStatusChanged: cfg.Kafka.Topics.MemberStatusChanged,
		},
	}, logger)
	if err != nil {
		logger.Fatal("failed to create kafka producer", zap.Error(err))
	}
	defer producer.Close()
	logger.Info("kafka producer initialized", zap.String("brokers", cfg.Kafka.Brokers))

	// ---- Application Layer ----
	repo := postgres.NewRepository(db)
	svc := appMember.NewService(repo, producer, logger)

	// ---- gRPC Server ----
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
	if err != nil {
		logger.Fatal("failed to listen on gRPC port", zap.Int("port", cfg.Server.GRPCPort), zap.Error(err))
	}

	grpcServer := grpc.NewServer(
		grpc.MaxConcurrentStreams(uint32(cfg.Server.MaxConcurrentRPC)),
	)

	// Register member service
	memberv1.RegisterMemberServiceServer(grpcServer, grpcHandler.NewMemberHandler(svc, logger))

	// Register gRPC health check
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("healthcare.member.v1.MemberService", grpc_health_v1.HealthCheckResponse_SERVING)

	// Register reflection for grpcurl / debugging
	reflection.Register(grpcServer)

	// ---- HTTP Mux (health + metrics + gRPC-gateway) ----
	httpMux := http.NewServeMux()

	// Health endpoints
	httpHandler.RegisterHealthRoutes(httpMux, nil) // pass nil checkers for liveness; extend for readiness

	// Prometheus metrics endpoint
	if cfg.Metrics.Enabled {
		httpMux.Handle(cfg.Metrics.Path, promhttp.Handler())
	}

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.HTTPPort),
		Handler:      httpMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ---- Start servers ----
	go func() {
		logger.Info("gRPC server starting", zap.Int("port", cfg.Server.GRPCPort))
		if err := grpcServer.Serve(grpcListener); err != nil {
			logger.Error("gRPC server error", zap.Error(err))
		}
	}()

	go func() {
		logger.Info("HTTP server starting",
			zap.Int("port", cfg.Server.HTTPPort),
			zap.String("metrics_path", cfg.Metrics.Path),
		)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", zap.Error(err))
		}
	}()

	// ---- Wait for shutdown signal ----
	<-ctx.Done()
	logger.Info("shutdown signal received")

	// ---- Graceful shutdown ----
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	// Stop health reporting so load balancers drain connections.
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// Stop gRPC: drains in-flight RPCs before closing.
	grpcServer.GracefulStop()
	logger.Info("gRPC server stopped")

	// Stop HTTP server.
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	}
	logger.Info("HTTP server stopped")

	logger.Info("shutdown complete")
}
