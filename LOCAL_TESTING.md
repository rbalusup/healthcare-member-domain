# Local End-to-End Testing Guide

This guide walks you through running and testing the Healthcare Member Service on your MacBook.

---

## Prerequisites

Install the following tools (if not already installed):

```bash
# Homebrew (if not installed)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Go 1.22+
brew install go

# Docker Desktop  ← runs PostgreSQL, Kafka, Prometheus, Grafana
# Download from: https://www.docker.com/products/docker-desktop/

# golang-migrate (for running DB migrations manually)
brew install golang-migrate

# grpcurl (for calling gRPC endpoints from the terminal)
brew install grpcurl

# Optional: GUI REST client
brew install --cask postman
```

Verify:

```bash
go version        # go1.22+
docker --version  # Docker 24+
grpcurl --version
migrate --version
```

---

## Option A — Full Stack via Docker Compose (Recommended)

This starts everything: PostgreSQL, Kafka, the member service, Prometheus, and Grafana — all wired together.

### 1. Start the stack

```bash
cd /path/to/healthcare-member-domain
make docker-up
```

Wait ~30 seconds for Kafka to become healthy (it has a 30s start period).

Check all containers are running:

```bash
docker compose -f build/docker-compose.yml ps
```

All services should show `healthy` or `running`.

### 2. Watch service logs

```bash
make docker-logs
# or
docker compose -f build/docker-compose.yml logs -f
```

### 3. Stop the stack

```bash
make docker-down
```

---

## Option B — Run Service Locally (Binary) Against Docker Dependencies

Use this when you want fast iteration — edit Go code and restart without rebuilding the Docker image.

### 1. Start only the infrastructure (no member-service container)

```bash
docker compose -f build/docker-compose.yml up postgres kafka -d
```

Wait for both to be healthy:

```bash
docker compose -f build/docker-compose.yml ps
```

### 2. Apply DB migrations

```bash
migrate \
  -path ./migrations \
  -database "postgres://member_user:member_pass@localhost:5432/member_db?sslmode=disable" \
  up
```

### 3. Run the service binary

```bash
export MEMBER_DB_HOST=localhost
export MEMBER_DB_PORT=5432
export MEMBER_DB_NAME=member_db
export MEMBER_DB_USER=member_user
export MEMBER_DB_PASSWORD=member_pass
export MEMBER_DB_SSL_MODE=disable
export MEMBER_KAFKA_BROKERS=localhost:9092
export MEMBER_LOG_LEVEL=debug
export MEMBER_LOG_FORMAT=console

make run
```

The service starts on:
- `localhost:9090` — gRPC
- `localhost:8080` — HTTP (gRPC-gateway REST)
- `localhost:2112/metrics` — Prometheus

---

## Running Unit Tests

No infrastructure needed — these are pure in-memory tests.

```bash
# All tests
make test

# With race detector
make test-race

# Open coverage report in browser
make cover
```

---

## Calling the API

### Via HTTP (REST — gRPC-gateway)

#### Create a member

```bash
curl -s -X POST http://localhost:8080/api/v1/members \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "Jane",
    "last_name": "Doe",
    "email": "jane.doe@example.com",
    "phone": "555-1234",
    "date_of_birth": "1985-06-15T00:00:00Z",
    "gender": "GENDER_FEMALE",
    "address": {
      "street": "123 Main St",
      "city": "Austin",
      "state": "TX",
      "zip_code": "78701",
      "country": "US"
    },
    "plan_id": "PLAN-001",
    "enrollment_date": "2026-01-01T00:00:00Z"
  }' | jq .
```

#### Get a member (replace `<member_id>` with the UUID from create response)

```bash
curl -s http://localhost:8080/api/v1/members/<member_id> | jq .
```

#### List members

```bash
curl -s "http://localhost:8080/api/v1/members?page=1&page_size=10" | jq .
```

#### Update a member

```bash
curl -s -X PUT http://localhost:8080/api/v1/members/<member_id> \
  -H "Content-Type: application/json" \
  -d '{"phone": "555-9999"}' | jq .
```

#### Enroll a member in a plan

```bash
curl -s -X POST http://localhost:8080/api/v1/members/<member_id>/enroll \
  -H "Content-Type: application/json" \
  -d '{
    "plan_id": "PLAN-002",
    "enrollment_date": "2026-03-01T00:00:00Z"
  }' | jq .
```

#### Get risk profile

```bash
curl -s http://localhost:8080/api/v1/members/<member_id>/risk-profile | jq .
```

---

### Via gRPC (grpcurl)

```bash
# List available services
grpcurl -plaintext localhost:9090 list

# List methods on MemberService
grpcurl -plaintext localhost:9090 list healthcare.member.v1.MemberService

# Create a member
grpcurl -plaintext -d '{
  "first_name": "John",
  "last_name": "Smith",
  "email": "john.smith@example.com",
  "plan_id": "PLAN-001",
  "enrollment_date": "2026-01-01T00:00:00Z",
  "date_of_birth": "1980-03-20T00:00:00Z"
}' localhost:9090 healthcare.member.v1.MemberService/CreateMember

# Get a member
grpcurl -plaintext -d '{"member_id": "<member_id>"}' \
  localhost:9090 healthcare.member.v1.MemberService/GetMember

# List members
grpcurl -plaintext -d '{"page": 1, "page_size": 10}' \
  localhost:9090 healthcare.member.v1.MemberService/ListMembers
```

---

## Health Check

```bash
curl http://localhost:8080/healthz
```

Expected: `{"status":"ok"}`

---

## Metrics

```bash
curl http://localhost:2112/metrics
```

Or open in browser: [http://localhost:2112/metrics](http://localhost:2112/metrics)

---

## Observability UIs

| Service    | URL                          | Credentials  |
|------------|------------------------------|--------------|
| Prometheus | http://localhost:9091        | none         |
| Grafana    | http://localhost:3000        | admin / admin |

In Grafana, dashboards are pre-provisioned from `monitoring/grafana/dashboards/`.

---

## Inspect the Database Directly

```bash
docker exec -it $(docker compose -f build/docker-compose.yml ps -q postgres) \
  psql -U member_user -d member_db

# Inside psql:
\dt                         -- list tables
SELECT * FROM members;      -- see all members
\q                          -- quit
```

---

## Troubleshooting

| Symptom | Fix |
|---|---|
| `kafka: topic not found` | Wait 30s for Kafka to fully start; check `docker compose ps` |
| `connection refused :9090` | Service not started; check `make docker-logs` |
| `pq: password authentication failed` | Ensure env vars are set (Option B) or Docker Compose is used |
| `migrate: no change` | Migrations already applied — safe to ignore |
| Build fails with CGO error | Install Xcode tools: `xcode-select --install` |
