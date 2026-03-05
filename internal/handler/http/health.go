package http

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"
)

// HealthChecker performs readiness checks on critical dependencies.
type HealthChecker interface {
	// Ping verifies the dependency is reachable (e.g., database ping).
	Ping(ctx context.Context) error
}

// HealthResponse is the JSON body returned by the health endpoints.
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Version   string            `json:"version"`
	Checks    map[string]string `json:"checks,omitempty"`
}

// HealthHandler returns an http.Handler that responds to liveness probes.
// It always returns 200 OK as long as the process is running.
func HealthHandler() http.HandlerFunc {
	version := os.Getenv("APP_VERSION")
	if version == "" {
		version = "development"
	}

	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, HealthResponse{
			Status:    "ok",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Version:   version,
		})
	}
}

// ReadinessHandler returns an http.Handler for readiness probes.
// It runs all provided HealthCheckers and returns 503 if any fail.
func ReadinessHandler(checkers map[string]HealthChecker) http.HandlerFunc {
	version := os.Getenv("APP_VERSION")
	if version == "" {
		version = "development"
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		status := http.StatusOK
		checks := make(map[string]string, len(checkers))

		for name, checker := range checkers {
			if err := checker.Ping(ctx); err != nil {
				checks[name] = "unhealthy: " + err.Error()
				status = http.StatusServiceUnavailable
			} else {
				checks[name] = "healthy"
			}
		}

		overallStatus := "ok"
		if status != http.StatusOK {
			overallStatus = "degraded"
		}

		writeJSON(w, status, HealthResponse{
			Status:    overallStatus,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Version:   version,
			Checks:    checks,
		})
	}
}

// RegisterHealthRoutes attaches health endpoints to the given ServeMux.
func RegisterHealthRoutes(mux *http.ServeMux, checkers map[string]HealthChecker) {
	mux.HandleFunc("/health", HealthHandler())   // Kubernetes liveness probe
	mux.HandleFunc("/healthz", HealthHandler())  // Alternative liveness path
	mux.HandleFunc("/ready", ReadinessHandler(checkers)) // Kubernetes readiness probe
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
