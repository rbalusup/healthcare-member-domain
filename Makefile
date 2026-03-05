# ──────────────────────────────────────────────
# Healthcare Member Service — Makefile
# ──────────────────────────────────────────────

BINARY     := member-service
CMD        := ./cmd/member-service
BUILD_DIR  := ./bin
DOCKER_IMG := healthcare/member-service

GO         := go
GOFLAGS    ?=
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "development")
LDFLAGS    := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: all build test test-race test-unit test-integration \
        lint lint-fix fmt vet generate \
        migrate-up migrate-down migrate-version \
        docker-build docker-push \
        run clean help

all: lint test build

# ──────────────────────────────────────────────
# Build
# ──────────────────────────────────────────────

## build: Compile the member-service binary
build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) $(CMD)
	@echo "Binary: $(BUILD_DIR)/$(BINARY)"

## run: Build and run the service locally
run: build
	$(BUILD_DIR)/$(BINARY)

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out

# ──────────────────────────────────────────────
# Test
# ──────────────────────────────────────────────

## test: Run all tests with coverage
test:
	$(GO) test $(GOFLAGS) -coverprofile=coverage.out -covermode=atomic ./...
	@$(GO) tool cover -func=coverage.out | grep -E "^total"

## test-race: Run all tests with race detector
test-race:
	$(GO) test $(GOFLAGS) -race -coverprofile=coverage.out -covermode=atomic ./...

## test-unit: Run only unit tests (skip integration)
test-unit:
	$(GO) test $(GOFLAGS) -short ./...

## test-integration: Run only integration tests
test-integration:
	$(GO) test $(GOFLAGS) -run Integration ./...

## cover: Open coverage report in browser
cover: test
	$(GO) tool cover -html=coverage.out

# ──────────────────────────────────────────────
# Code Quality
# ──────────────────────────────────────────────

## lint: Run golangci-lint
lint:
	golangci-lint run --timeout 5m ./...

## lint-fix: Run golangci-lint with auto-fix
lint-fix:
	golangci-lint run --fix --timeout 5m ./...

## fmt: Format Go source files
fmt:
	$(GO) fmt ./...
	goimports -w .

## vet: Run go vet
vet:
	$(GO) vet ./...

# ──────────────────────────────────────────────
# Code Generation
# ──────────────────────────────────────────────

## generate: Run protoc to generate Go code from .proto files
generate:
	./scripts/generate.sh

# ──────────────────────────────────────────────
# Database Migrations
# ──────────────────────────────────────────────

## migrate-up: Apply all pending migrations
migrate-up:
	./scripts/migrate.sh up

## migrate-down: Roll back one migration
migrate-down:
	./scripts/migrate.sh down 1

## migrate-version: Show current migration version
migrate-version:
	./scripts/migrate.sh version

# ──────────────────────────────────────────────
# Docker
# ──────────────────────────────────────────────

## docker-build: Build the Docker image
docker-build:
	docker build \
		--build-arg VERSION=$(VERSION) \
		-f build/Dockerfile \
		-t $(DOCKER_IMG):$(VERSION) \
		-t $(DOCKER_IMG):latest \
		.

## docker-push: Push the Docker image to registry
docker-push: docker-build
	docker push $(DOCKER_IMG):$(VERSION)
	docker push $(DOCKER_IMG):latest

## docker-up: Start the full local dev stack
docker-up:
	docker compose -f build/docker-compose.yml up --build -d
	@echo "Services started:"
	@echo "  gRPC:       localhost:9090"
	@echo "  HTTP:       localhost:8080"
	@echo "  Metrics:    localhost:2112/metrics"
	@echo "  Prometheus: localhost:9091"
	@echo "  Grafana:    localhost:3000 (admin/admin)"

## docker-down: Stop the local dev stack
docker-down:
	docker compose -f build/docker-compose.yml down

## docker-logs: Tail member-service logs
docker-logs:
	docker compose -f build/docker-compose.yml logs -f member-service

# ──────────────────────────────────────────────
# Utilities
# ──────────────────────────────────────────────

## deps: Download and tidy Go modules
deps:
	$(GO) mod download
	$(GO) mod tidy

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | \
		sed 's/## //' | \
		awk -F: '{ printf "  %-22s %s\n", $$1, $$2 }'
