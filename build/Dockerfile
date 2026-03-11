# ──────────────────────────────────────────────
# Stage 1: Builder
# ──────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

# Install build dependencies for confluent-kafka-go (CGo required)
RUN apk add --no-cache gcc musl-dev librdkafka-dev pkgconf

WORKDIR /app

# Download dependencies first (cached layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build the binary with CGO enabled for confluent-kafka-go
RUN CGO_ENABLED=1 GOOS=linux \
    go build \
    -ldflags="-s -w -X main.version=${VERSION:-development}" \
    -o /bin/member-service \
    ./cmd/member-service

# ──────────────────────────────────────────────
# Stage 2: Runtime
# ──────────────────────────────────────────────
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    librdkafka \
    tzdata && \
    addgroup -S appgroup && \
    adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /bin/member-service .

# Non-root user
USER appuser

# Ports:
#   9090 — gRPC
#   8080 — HTTP (health + gRPC-gateway)
#   2112 — Prometheus metrics
EXPOSE 9090 8080 2112

HEALTHCHECK --interval=15s --timeout=5s --start-period=20s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["./member-service"]
