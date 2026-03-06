package postgres

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// HealthChecker verifies PostgreSQL connectivity.
type HealthChecker struct {
	db *sqlx.DB
}

// NewHealthChecker returns a HealthChecker backed by the given connection pool.
func NewHealthChecker(db *sqlx.DB) *HealthChecker {
	return &HealthChecker{db: db}
}

// Ping sends a lightweight ping to the database and returns any error.
func (h *HealthChecker) Ping(ctx context.Context) error {
	return h.db.PingContext(ctx)
}
