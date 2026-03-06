package kafka

import (
	"context"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// HealthChecker verifies Kafka broker connectivity.
type HealthChecker struct {
	brokers string
}

// NewHealthChecker returns a HealthChecker that probes the given Kafka brokers.
func NewHealthChecker(brokers string) *HealthChecker {
	return &HealthChecker{brokers: brokers}
}

// Ping creates a short-lived admin client and fetches cluster metadata.
// A 3-second timeout is applied via the metadata request timeout.
func (h *HealthChecker) Ping(_ context.Context) error {
	admin, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": h.brokers,
		"socket.timeout.ms": 3000,
	})
	if err != nil {
		return fmt.Errorf("creating kafka admin client: %w", err)
	}
	defer admin.Close()

	_, err = admin.GetMetadata(nil, true, 3000)
	if err != nil {
		return fmt.Errorf("kafka metadata fetch failed: %w", err)
	}
	return nil
}
