package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config is the top-level application configuration.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Kafka    KafkaConfig    `mapstructure:"kafka"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
	Log      LogConfig      `mapstructure:"log"`
}

// ServerConfig holds gRPC and HTTP gateway listener settings.
type ServerConfig struct {
	GRPCPort         int           `mapstructure:"grpc_port"`
	HTTPPort         int           `mapstructure:"http_port"`
	ShutdownTimeout  time.Duration `mapstructure:"shutdown_timeout"`
	MaxConcurrentRPC int           `mapstructure:"max_concurrent_rpc"`
}

// DatabaseConfig holds PostgreSQL connection parameters.
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Name            string        `mapstructure:"name"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// DSN returns the PostgreSQL connection string.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

// KafkaConfig holds Kafka broker and topic configuration.
type KafkaConfig struct {
	Brokers         string `mapstructure:"brokers"`
	GroupID         string `mapstructure:"group_id"`
	TransactionalID string `mapstructure:"transactional_id"`
	Topics          KafkaTopics `mapstructure:"topics"`
}

// KafkaTopics defines the topic names this service produces and consumes.
type KafkaTopics struct {
	MemberCreated       string `mapstructure:"member_created"`
	MemberUpdated       string `mapstructure:"member_updated"`
	MemberEnrolled      string `mapstructure:"member_enrolled"`
	MemberStatusChanged string `mapstructure:"member_status_changed"`
}

// MetricsConfig holds Prometheus metrics server settings.
type MetricsConfig struct {
	Port    int    `mapstructure:"port"`
	Path    string `mapstructure:"path"`
	Enabled bool   `mapstructure:"enabled"`
}

// LogConfig holds structured logging settings.
type LogConfig struct {
	Level  string `mapstructure:"level"`  // debug, info, warn, error
	Format string `mapstructure:"format"` // json, console
}

// Load reads configuration from environment variables and optional config file.
// Environment variable naming: MEMBER_<SECTION>_<KEY> (e.g. MEMBER_DB_HOST).
func Load() (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("server.grpc_port", 9090)
	v.SetDefault("server.http_port", 8080)
	v.SetDefault("server.shutdown_timeout", 30*time.Second)
	v.SetDefault("server.max_concurrent_rpc", 1000)

	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.name", "member_db")
	v.SetDefault("database.user", "member_user")
	v.SetDefault("database.password", "")
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", 5*time.Minute)

	v.SetDefault("kafka.brokers", "localhost:9092")
	v.SetDefault("kafka.group_id", "member-service")
	v.SetDefault("kafka.transactional_id", "member-service-producer-1")
	v.SetDefault("kafka.topics.member_created", "member.created")
	v.SetDefault("kafka.topics.member_updated", "member.updated")
	v.SetDefault("kafka.topics.member_enrolled", "member.enrolled")
	v.SetDefault("kafka.topics.member_status_changed", "member.status.changed")

	v.SetDefault("metrics.port", 2112)
	v.SetDefault("metrics.path", "/metrics")
	v.SetDefault("metrics.enabled", true)

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")

	// Environment variable binding
	// All env vars prefixed with MEMBER_ and using _ as separator.
	v.SetEnvPrefix("MEMBER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Optional config file (config.yaml in working directory or /etc/member-service)
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("/etc/member-service")
	// Ignore missing config file — env vars are sufficient.
	_ = v.ReadInConfig()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}
