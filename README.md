# Healthcare Member Service

A production-grade Go microservice for the **Member domain** of a healthcare platform that moves care delivery to lower-cost settings — primarily the home. The Member service is the unified platform orchestrating clinical data, scheduling, risk prediction, and provider networks for 1.5M+ members, 10,000+ mobile clinicians, and 3,500 value-based providers.

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                     Member Service                           │
│                                                              │
│  ┌─────────────┐   ┌──────────────┐   ┌──────────────────┐ │
│  │  gRPC/HTTP  │   │  Application │   │     Domain       │ │
│  │  Handlers   │──▶│   Service    │──▶│  Member Entity   │ │
│  └─────────────┘   └──────┬───────┘   └──────────────────┘ │
│                           │                                  │
│              ┌────────────┼────────────┐                     │
│              ▼            ▼            ▼                     │
│         ┌─────────┐ ┌─────────┐ ┌──────────┐               │
│         │ Postgres│ │  Kafka  │ │ Metrics  │               │
│         │  Repo   │ │Producer │ │Prometheus│               │
│         └─────────┘ └─────────┘ └──────────┘               │
└──────────────────────────────────────────────────────────────┘
```

### DDD Layer Mapping

| Layer | Package | Responsibility |
|-------|---------|---------------|
| Domain | `internal/domain/member` | Entities, value objects, repository/service interfaces, domain errors |
| Application | `internal/application/member` | Use cases, command/query DTOs, application service |
| Infrastructure | `internal/infrastructure/*` | PostgreSQL, Kafka EOS, Prometheus metrics |
| Handler | `internal/handler/*` | gRPC handlers, HTTP health endpoints |
| Config | `internal/config` | Viper-based configuration with env var binding |

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.22 |
| API | gRPC + Protocol Buffers v3 |
| REST bridge | gRPC-gateway v2 |
| Database | PostgreSQL 16 (sqlx + lib/pq) |
| Messaging | Kafka (confluent-kafka-go, exactly-once semantics) |
| Observability | Prometheus + Grafana |
| Logging | Uber zap (structured JSON) |
| Config | Viper (env vars + optional YAML file) |
| Container | Docker (multi-stage Alpine build) |
| CI/CD | GitHub Actions |
| Deploy | AWS ECS Fargate + GCP GKE |

## Service Ports

| Port | Protocol | Purpose |
|------|---------|---------|
| `9090` | gRPC | Member service gRPC API |
| `8080` | HTTP | Health endpoints + gRPC-gateway REST |
| `2112` | HTTP | Prometheus `/metrics` |

## Quick Start

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- `protoc` + Go proto plugins (for code generation)

### Run locally

```bash
# 1. Start infrastructure (Postgres, Kafka, Prometheus, Grafana)
make docker-up

# 2. Run database migrations
make migrate-up

# 3. Generate gRPC Go code from proto files
make generate

# 4. Build and run the service
make run
```

The service is now available at:
- gRPC: `localhost:9090`
- Health check: `http://localhost:8080/health`
- Prometheus metrics: `http://localhost:2112/metrics`
- Grafana dashboards: `http://localhost:3000` (admin/admin)

### Run tests

```bash
make test          # all tests with coverage
make test-race     # with race detector
make test-unit     # unit tests only (no infra required)
```

## API Reference

The gRPC service definition lives in [api/member/v1/member.proto](api/member/v1/member.proto).

### MemberService RPCs

| Method | HTTP | Description |
|--------|------|-------------|
| `CreateMember` | `POST /api/v1/members` | Create a new member |
| `GetMember` | `GET /api/v1/members/{member_id}` | Retrieve member by ID |
| `UpdateMember` | `PUT /api/v1/members/{member_id}` | Update member profile |
| `ListMembers` | `GET /api/v1/members` | Paginated list with filters |
| `EnrollMember` | `POST /api/v1/members/{member_id}/enroll` | Enroll member in a plan |
| `GetMemberRiskProfile` | `GET /api/v1/members/{member_id}/risk-profile` | Get risk score and care level |

### Example: Create Member (gRPCurl)

```bash
grpcurl -plaintext -d '{
  "first_name": "Jane",
  "last_name": "Doe",
  "email": "jane.doe@example.com",
  "plan_id": "PPO-2024-001",
  "enrollment_date": "2024-01-01T00:00:00Z",
  "date_of_birth": "1985-06-15T00:00:00Z"
}' localhost:9090 healthcare.member.v1.MemberService/CreateMember
```

## Kafka Events

| Topic | Trigger |
|-------|---------|
| `member.created` | New member record created |
| `member.updated` | Member profile updated |
| `member.enrolled` | Member enrolled in a plan |
| `member.status.changed` | Member status transition |

All events use exactly-once semantics (Kafka transactions + idempotent producer).

## Configuration

All configuration is via environment variables prefixed with `MEMBER_`.
An optional `config.yaml` file in the working directory or `/etc/member-service/` is also supported.

| Variable | Default | Description |
|----------|---------|-------------|
| `MEMBER_SERVER_GRPC_PORT` | `9090` | gRPC listen port |
| `MEMBER_SERVER_HTTP_PORT` | `8080` | HTTP listen port |
| `MEMBER_SERVER_SHUTDOWN_TIMEOUT` | `30s` | Graceful shutdown timeout |
| `MEMBER_DB_HOST` | `localhost` | PostgreSQL host |
| `MEMBER_DB_PORT` | `5432` | PostgreSQL port |
| `MEMBER_DB_NAME` | `member_db` | Database name |
| `MEMBER_DB_USER` | `member_user` | Database user |
| `MEMBER_DB_PASSWORD` | _(required)_ | Database password |
| `MEMBER_DB_SSL_MODE` | `disable` | PostgreSQL SSL mode |
| `MEMBER_KAFKA_BROKERS` | `localhost:9092` | Kafka bootstrap servers |
| `MEMBER_KAFKA_GROUP_ID` | `member-service` | Kafka consumer group |
| `MEMBER_METRICS_ENABLED` | `true` | Enable Prometheus metrics |
| `MEMBER_LOG_LEVEL` | `info` | Log level (debug/info/warn/error) |
| `MEMBER_LOG_FORMAT` | `json` | Log format (json/console) |

## Project Structure

```
.
├── api/member/v1/          # Proto definitions (source of truth for API)
├── gen/go/member/v1/       # Generated code (run: make generate)
├── cmd/member-service/     # main.go — entry point + wiring
├── internal/
│   ├── config/             # Configuration loading
│   ├── domain/member/      # Domain entities, interfaces, errors
│   ├── application/member/ # Use cases, commands, queries
│   ├── infrastructure/     # Postgres, Kafka, metrics implementations
│   └── handler/            # gRPC and HTTP handlers
├── migrations/             # SQL migration files
├── build/                  # Dockerfile + docker-compose.yml
├── deploy/                 # AWS (ECS) and GCP (GKE) manifests
├── monitoring/             # Prometheus config + Grafana dashboards
├── scripts/                # generate.sh, migrate.sh
└── .github/workflows/      # CI (test+lint+build) and CD (deploy)
```

## Makefile Targets

```
make build           Compile the binary
make run             Build and run locally
make test            Run tests with coverage
make test-race       Run tests with race detector
make lint            golangci-lint
make generate        Generate Go code from proto files
make migrate-up      Apply database migrations
make migrate-down    Roll back last migration
make docker-build    Build Docker image
make docker-up       Start full local stack
make docker-down     Stop local stack
make help            Show all targets
```

## Deployment

### AWS (ECS Fargate)
- Task definition: [deploy/aws/task-definition.json](deploy/aws/task-definition.json)
- Secrets injected via AWS SSM Parameter Store
- Container image: pushed to Amazon ECR via GitHub Actions CD workflow

### GCP (GKE)
- Deployment manifest: [deploy/gcp/deployment.yaml](deploy/gcp/deployment.yaml)
- Includes HorizontalPodAutoscaler (3–30 replicas, 70% CPU target)
- Secrets injected via Kubernetes Secrets
- Container image: pushed to GCR via GitHub Actions CD workflow

## Contributing

1. Branch from `develop`
2. `make lint test` must pass before opening a PR
3. PR must target `develop`; merges to `main` trigger production deployment
