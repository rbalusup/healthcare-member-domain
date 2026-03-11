package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	memberv1 "github.com/rbalusup/healthcare-member-domain/gen/go/member/v1"
	appMember "github.com/rbalusup/healthcare-member-domain/internal/application/member"
	"github.com/rbalusup/healthcare-member-domain/internal/config"
	grpcHandler "github.com/rbalusup/healthcare-member-domain/internal/handler/grpc"
	httpHandler "github.com/rbalusup/healthcare-member-domain/internal/handler/http"
	"github.com/rbalusup/healthcare-member-domain/internal/infrastructure/kafka"
	"github.com/rbalusup/healthcare-member-domain/internal/infrastructure/metrics"
	middleware "github.com/rbalusup/healthcare-member-domain/internal/middleware/grpc"
	"github.com/rbalusup/healthcare-member-domain/internal/infrastructure/postgres"
)

func main() {
	// ---- Signal context ----
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

	// ---- Kafka Consumer ----
	consumer, err := kafka.NewConsumer(kafka.ConsumerConfig{
		Brokers:     cfg.Kafka.Brokers,
		GroupID:     cfg.Kafka.GroupID,
		Topics:      []string{"member.risk.updated", "member.status.external"},
		PollTimeout: 100 * time.Millisecond,
	}, logger)
	if err != nil {
		logger.Fatal("failed to create kafka consumer", zap.Error(err))
	}

	// ---- Application Layer ----
	repo := postgres.NewRepository(db)
	svc := appMember.NewService(repo, producer, logger, m)

	// ---- gRPC Server ----
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
	if err != nil {
		logger.Fatal("failed to listen on gRPC port", zap.Int("port", cfg.Server.GRPCPort), zap.Error(err))
	}

	grpcServer := grpc.NewServer(
		grpc.MaxConcurrentStreams(uint32(cfg.Server.MaxConcurrentRPC)),
		grpc.ChainUnaryInterceptor(
			middleware.RequestIDInterceptor(),
			middleware.LoggingInterceptor(logger),
			middleware.MetricsInterceptor(m),
			middleware.RecoveryInterceptor(logger),
		),
	)

	memberv1.RegisterMemberServiceServer(grpcServer, grpcHandler.NewMemberHandler(svc, logger))

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("healthcare.member.v1.MemberService", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(grpcServer)

	// ---- gRPC-Gateway REST Bridge ----
	gwMux := runtime.NewServeMux()
	gwConn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%d", cfg.Server.GRPCPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logger.Fatal("failed to create gateway gRPC connection", zap.Error(err))
	}
	if err := memberv1.RegisterMemberServiceHandlerClient(ctx, gwMux, memberv1.NewMemberServiceClient(gwConn)); err != nil {
		logger.Fatal("failed to register gRPC-gateway handler", zap.Error(err))
	}

	// ---- HTTP Mux (health + metrics + gRPC-gateway) ----
	httpMux := http.NewServeMux()

	httpHandler.RegisterHealthRoutes(httpMux, map[string]httpHandler.HealthChecker{
		"postgres": postgres.NewHealthChecker(db),
		"kafka":    kafka.NewHealthChecker(cfg.Kafka.Brokers),
	})

	httpMux.Handle("/api/", gwMux)

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

	// ---- Kafka Consumer goroutine ----
	go func() {
		if err := consumer.Run(ctx, memberEventHandler(logger)); err != nil {
			logger.Error("kafka consumer stopped", zap.Error(err))
		}
	}()

	// ---- Wait for shutdown signal ----
	<-ctx.Done()
	logger.Info("shutdown signal received")

	// ---- Graceful shutdown ----
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	grpcServer.GracefulStop()
	logger.Info("gRPC server stopped")

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	}
	logger.Info("HTTP server stopped")

	logger.Info("shutdown complete")
}

// memberEventHandler routes consumed Kafka events to the appropriate use cases.
func memberEventHandler(logger *zap.Logger) kafka.MessageHandler {
	return func(ctx context.Context, topic string, key, value []byte) error {
		event, err := kafka.ParseEvent(value)
		if err != nil {
			logger.Warn("failed to parse kafka event",
				zap.String("topic", topic),
				zap.Error(err),
			)
			return nil // non-retryable parse error — log and skip
		}

		logger.Info("kafka event received",
			zap.String("topic", topic),
			zap.String("event_type", event.EventType),
			zap.String("member_id", event.MemberID),
		)

		// Route to use-case handlers based on event type.
		// Additional routing (e.g., risk score update from ML pipeline) goes here.
		var payload map[string]interface{}
		if data, err := json.Marshal(event.Payload); err == nil {
			_ = json.Unmarshal(data, &payload)
		}

		return nil
	}
}
