package kafka

import "time"

// MemberEvent is the envelope published to Kafka for all member domain events.
type MemberEvent struct {
	EventType string      `json:"event_type"`
	MemberID  string      `json:"member_id"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}
