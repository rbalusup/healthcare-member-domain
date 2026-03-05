package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"

	"github.com/healthcare/member-service/internal/domain/member"
)

// MemberEvent is the envelope published to Kafka for all member domain events.
type MemberEvent struct {
	EventType string      `json:"event_type"`
	MemberID  string      `json:"member_id"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// Producer publishes member domain events to Kafka using exactly-once semantics.
type Producer struct {
	producer *kafka.Producer
	topics   Topics
	logger   *zap.Logger
}

// Topics holds the Kafka topic names for each event type.
type Topics struct {
	MemberCreated       string
	MemberUpdated       string
	MemberEnrolled      string
	MemberStatusChanged string
}

// ProducerConfig holds configuration for the Kafka producer.
type ProducerConfig struct {
	Brokers         string
	TransactionalID string
	Topics          Topics
}

// NewProducer constructs a Kafka producer with exactly-once semantics enabled.
// The producer uses idempotent delivery and transactional API to guarantee
// that each message is delivered exactly once to the broker.
func NewProducer(cfg ProducerConfig, logger *zap.Logger) (*Producer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":                     cfg.Brokers,
		"transactional.id":                      cfg.TransactionalID,
		"enable.idempotence":                    true,
		"acks":                                  "all",
		"retries":                               2147483647,
		"max.in.flight.requests.per.connection": 5,
		"message.timeout.ms":                    30000,
	})
	if err != nil {
		return nil, fmt.Errorf("creating kafka producer: %w", err)
	}

	// Initialize transactions — required before BeginTransaction.
	if err := p.InitTransactions(context.Background()); err != nil {
		p.Close()
		return nil, fmt.Errorf("initializing kafka transactions: %w", err)
	}

	return &Producer{
		producer: p,
		topics:   cfg.Topics,
		logger:   logger,
	}, nil
}

// PublishMemberCreated publishes a member.created event.
func (p *Producer) PublishMemberCreated(ctx context.Context, m *member.Member) error {
	return p.publish(ctx, p.topics.MemberCreated, "member.created", m.MemberID.String(), m)
}

// PublishMemberUpdated publishes a member.updated event.
func (p *Producer) PublishMemberUpdated(ctx context.Context, m *member.Member) error {
	return p.publish(ctx, p.topics.MemberUpdated, "member.updated", m.MemberID.String(), m)
}

// PublishMemberEnrolled publishes a member.enrolled event.
func (p *Producer) PublishMemberEnrolled(ctx context.Context, m *member.Member) error {
	return p.publish(ctx, p.topics.MemberEnrolled, "member.enrolled", m.MemberID.String(), m)
}

// PublishMemberStatusChanged publishes a member.status.changed event.
func (p *Producer) PublishMemberStatusChanged(ctx context.Context, m *member.Member, previous member.MemberStatus) error {
	payload := map[string]interface{}{
		"member":          m,
		"previous_status": previous,
	}
	return p.publish(ctx, p.topics.MemberStatusChanged, "member.status.changed", m.MemberID.String(), payload)
}

// publish wraps a single message in a Kafka transaction.
// For production workloads with transactional consumers, the consumer offsets
// should be committed inside the same transaction using SendOffsetsToTransaction.
func (p *Producer) publish(ctx context.Context, topic, eventType, key string, payload interface{}) error {
	event := MemberEvent{
		EventType: eventType,
		MemberID:  key,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}

	value, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling event: %w", err)
	}

	if err := p.producer.BeginTransaction(); err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	deliveryChan := make(chan kafka.Event, 1)
	err = p.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Key:   []byte(key),
		Value: value,
	}, deliveryChan)
	if err != nil {
		_ = p.producer.AbortTransaction(ctx)
		return fmt.Errorf("producing message: %w", err)
	}

	// Wait for delivery report.
	select {
	case e := <-deliveryChan:
		msg, ok := e.(*kafka.Message)
		if !ok || msg.TopicPartition.Error != nil {
			_ = p.producer.AbortTransaction(ctx)
			if ok {
				return fmt.Errorf("message delivery failed: %w", msg.TopicPartition.Error)
			}
			return fmt.Errorf("unexpected kafka event type")
		}
	case <-ctx.Done():
		_ = p.producer.AbortTransaction(ctx)
		return ctx.Err()
	}

	if err := p.producer.CommitTransaction(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	p.logger.Debug("kafka event published",
		zap.String("topic", topic),
		zap.String("event_type", eventType),
		zap.String("key", key),
	)
	return nil
}

// Close flushes pending messages and closes the producer.
func (p *Producer) Close() {
	p.producer.Flush(15000) // 15s flush timeout
	p.producer.Close()
}
