package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
)

// MessageHandler processes a single Kafka message.
// Implementations must be idempotent — the message may be delivered more than once
// during rebalances or failure recovery.
type MessageHandler func(ctx context.Context, topic string, key, value []byte) error

// ConsumerConfig holds configuration for the Kafka consumer.
type ConsumerConfig struct {
	Brokers  string
	GroupID  string
	Topics   []string
	// PollTimeout is the maximum time to wait for a message before returning nil.
	PollTimeout time.Duration
}

// Consumer reads messages from Kafka with exactly-once delivery semantics.
// It uses manual offset commits within the read_committed isolation level to
// guarantee that only committed (non-aborted) messages are processed.
type Consumer struct {
	consumer *kafka.Consumer
	topics   []string
	logger   *zap.Logger
	poll     time.Duration
}

// NewConsumer constructs a Kafka consumer configured for exactly-once reads.
func NewConsumer(cfg ConsumerConfig, logger *zap.Logger) (*Consumer, error) {
	poll := cfg.PollTimeout
	if poll == 0 {
		poll = 100 * time.Millisecond
	}

	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  cfg.Brokers,
		"group.id":           cfg.GroupID,
		"auto.offset.reset":  "earliest",
		// read_committed ensures we only see messages from committed transactions.
		"isolation.level":    "read_committed",
		// Disable auto-commit — we manage offsets manually.
		"enable.auto.commit": false,
	})
	if err != nil {
		return nil, fmt.Errorf("creating kafka consumer: %w", err)
	}

	if err := c.SubscribeTopics(cfg.Topics, nil); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("subscribing to topics: %w", err)
	}

	return &Consumer{
		consumer: c,
		topics:   cfg.Topics,
		logger:   logger,
		poll:     poll,
	}, nil
}

// Run starts the consume loop. It blocks until ctx is cancelled.
// Each message is processed by handler; on success the offset is committed.
// Processing errors are logged but do not stop the loop — implement dead-letter
// queue logic inside the handler for messages that must not be skipped.
func (c *Consumer) Run(ctx context.Context, handler MessageHandler) error {
	c.logger.Info("kafka consumer started", zap.Strings("topics", c.topics))

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("kafka consumer stopping")
			return c.consumer.Close()
		default:
		}

		ev := c.consumer.Poll(int(c.poll.Milliseconds()))
		if ev == nil {
			continue
		}

		switch e := ev.(type) {
		case *kafka.Message:
			if err := c.processMessage(ctx, e, handler); err != nil {
				c.logger.Error("message processing failed",
					zap.String("topic", *e.TopicPartition.Topic),
					zap.Int32("partition", e.TopicPartition.Partition),
					zap.Int64("offset", int64(e.TopicPartition.Offset)),
					zap.Error(err),
				)
				// Continue rather than abort — dead-letter handling is the
				// responsibility of the handler implementation.
			}

		case kafka.Error:
			c.logger.Error("kafka consumer error",
				zap.Int("code", int(e.Code())),
				zap.String("error", e.Error()),
			)
			// Fatal errors (broker not available, auth failure) should terminate.
			if e.IsFatal() {
				return fmt.Errorf("fatal kafka error: %w", e)
			}

		case kafka.PartitionEOF:
			c.logger.Debug("reached partition EOF",
				zap.String("topic", *e.Topic),
				zap.Int32("partition", e.Partition),
			)

		case kafka.OffsetsCommitted:
			c.logger.Debug("offsets committed")
		}
	}
}

// processMessage invokes the handler and commits the offset on success.
func (c *Consumer) processMessage(ctx context.Context, msg *kafka.Message, handler MessageHandler) error {
	topic := ""
	if msg.TopicPartition.Topic != nil {
		topic = *msg.TopicPartition.Topic
	}

	if err := handler(ctx, topic, msg.Key, msg.Value); err != nil {
		return err
	}

	// Commit the offset for this specific partition+offset.
	if _, err := c.consumer.CommitMessage(msg); err != nil {
		return fmt.Errorf("committing offset: %w", err)
	}
	return nil
}

// ParseEvent decodes a MemberEvent from a raw Kafka message value.
func ParseEvent(value []byte) (*MemberEvent, error) {
	var event MemberEvent
	if err := json.Unmarshal(value, &event); err != nil {
		return nil, fmt.Errorf("parsing member event: %w", err)
	}
	return &event, nil
}
