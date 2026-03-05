package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus instruments for the member service.
// Use promauto to auto-register with the default registry.
type Metrics struct {
	// Business counters
	MembersCreatedTotal      prometheus.Counter
	MembersUpdatedTotal      prometheus.Counter
	MembersEnrolledTotal     prometheus.Counter
	MembersTerminatedTotal   prometheus.Counter

	// Error counters
	MembersCreatedErrorsTotal  prometheus.Counter
	MembersUpdatedErrorsTotal  prometheus.Counter

	// Request latency histograms (gRPC method level)
	GRPCRequestDuration *prometheus.HistogramVec

	// Database query latency
	DBQueryDuration *prometheus.HistogramVec

	// Kafka
	KafkaMessagesPublishedTotal  *prometheus.CounterVec
	KafkaMessagesConsumedTotal   *prometheus.CounterVec
	KafkaPublishErrorsTotal      *prometheus.CounterVec

	// Active members gauge (updated periodically by a background goroutine)
	ActiveMembersTotal prometheus.Gauge
}

// New registers all metrics and returns the Metrics struct.
func New() *Metrics {
	latencyBuckets := []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5}

	return &Metrics{
		MembersCreatedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "member_service",
			Name:      "members_created_total",
			Help:      "Total number of member records created.",
		}),

		MembersUpdatedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "member_service",
			Name:      "members_updated_total",
			Help:      "Total number of member records updated.",
		}),

		MembersEnrolledTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "member_service",
			Name:      "members_enrolled_total",
			Help:      "Total number of successful member enrollments.",
		}),

		MembersTerminatedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "member_service",
			Name:      "members_terminated_total",
			Help:      "Total number of member enrollment terminations.",
		}),

		MembersCreatedErrorsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "member_service",
			Name:      "members_created_errors_total",
			Help:      "Total errors encountered while creating members.",
		}),

		MembersUpdatedErrorsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "member_service",
			Name:      "members_updated_errors_total",
			Help:      "Total errors encountered while updating members.",
		}),

		GRPCRequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "member_service",
			Name:      "grpc_request_duration_seconds",
			Help:      "Histogram of gRPC request latency in seconds.",
			Buckets:   latencyBuckets,
		}, []string{"method", "status"}),

		DBQueryDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "member_service",
			Name:      "db_query_duration_seconds",
			Help:      "Histogram of database query latency in seconds.",
			Buckets:   latencyBuckets,
		}, []string{"operation"}),

		KafkaMessagesPublishedTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "member_service",
			Name:      "kafka_messages_published_total",
			Help:      "Total Kafka messages published, by topic.",
		}, []string{"topic"}),

		KafkaMessagesConsumedTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "member_service",
			Name:      "kafka_messages_consumed_total",
			Help:      "Total Kafka messages consumed, by topic.",
		}, []string{"topic"}),

		KafkaPublishErrorsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "member_service",
			Name:      "kafka_publish_errors_total",
			Help:      "Total Kafka publish errors, by topic.",
		}, []string{"topic"}),

		ActiveMembersTotal: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "member_service",
			Name:      "active_members_total",
			Help:      "Current number of active members.",
		}),
	}
}
