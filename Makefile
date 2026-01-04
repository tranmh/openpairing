.PHONY: all build test lint clean dev docker-up docker-down migrate help

# Variables
BACKEND_DIR := backend
FRONTEND_DIR := frontend
BIN_DIR := bin

# Default target
all: lint test build

# Help
help:
	@echo "OpenPairing - Chess Tournament Management Platform"
	@echo ""
	@echo "Usage:"
	@echo "  make dev          - Start development environment"
	@echo "  make build        - Build all services"
	@echo "  make test         - Run all tests"
	@echo "  make lint         - Run linters"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make docker-up    - Start Docker services"
	@echo "  make docker-down  - Stop Docker services"
	@echo "  make migrate-up   - Run database migrations"
	@echo "  make migrate-down - Rollback database migrations"
	@echo ""

# Development
dev: docker-up
	@echo "Starting development servers..."
	@cd $(BACKEND_DIR) && go run ./cmd/api &
	@cd $(FRONTEND_DIR) && npm run dev

# Build targets
build: build-backend build-frontend

build-backend:
	@echo "Building backend services..."
	@mkdir -p $(BIN_DIR)
	cd $(BACKEND_DIR) && CGO_ENABLED=0 go build -ldflags="-w -s" -o ../$(BIN_DIR)/api ./cmd/api
	cd $(BACKEND_DIR) && CGO_ENABLED=0 go build -ldflags="-w -s" -o ../$(BIN_DIR)/chess ./cmd/chess
	cd $(BACKEND_DIR) && CGO_ENABLED=0 go build -ldflags="-w -s" -o ../$(BIN_DIR)/worker ./cmd/worker
	@echo "Backend built successfully"

build-frontend:
	@echo "Building frontend..."
	cd $(FRONTEND_DIR) && npm ci && npm run build
	@echo "Frontend built successfully"

# Test targets
test: test-backend test-frontend

test-backend:
	@echo "Running backend tests..."
	cd $(BACKEND_DIR) && go test -v -race -coverprofile=coverage.out ./...

test-frontend:
	@echo "Running frontend tests..."
	cd $(FRONTEND_DIR) && npm run test:coverage

# Lint targets
lint: lint-backend lint-frontend

lint-backend:
	@echo "Linting backend..."
	cd $(BACKEND_DIR) && golangci-lint run --timeout=5m

lint-frontend:
	@echo "Linting frontend..."
	cd $(FRONTEND_DIR) && npm run lint && npm run format:check && npm run typecheck

# Clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BIN_DIR)
	rm -rf $(FRONTEND_DIR)/.next
	rm -rf $(FRONTEND_DIR)/node_modules
	rm -rf $(BACKEND_DIR)/coverage.out
	rm -rf $(FRONTEND_DIR)/coverage

# Docker targets
docker-up:
	@echo "Starting Docker services..."
	docker-compose up -d postgres redis minio
	@echo "Waiting for services to be ready..."
	@sleep 5
	@echo "Services are ready"

docker-down:
	@echo "Stopping Docker services..."
	docker-compose down

docker-build:
	@echo "Building Docker images..."
	docker-compose build

docker-logs:
	docker-compose logs -f

# Database migrations
migrate-up:
	@echo "Running migrations..."
	migrate -path $(BACKEND_DIR)/migrations -database "postgres://openpairing:openpairing@localhost:5432/openpairing?sslmode=disable" up

migrate-down:
	@echo "Rolling back migrations..."
	migrate -path $(BACKEND_DIR)/migrations -database "postgres://openpairing:openpairing@localhost:5432/openpairing?sslmode=disable" down 1

migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir $(BACKEND_DIR)/migrations -seq $$name

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	cd $(FRONTEND_DIR) && npm install

# Format code
fmt:
	@echo "Formatting code..."
	cd $(BACKEND_DIR) && go fmt ./...
	cd $(FRONTEND_DIR) && npm run format

# Generate API docs
docs:
	@echo "Generating API documentation..."
	cd $(BACKEND_DIR) && swag init -g cmd/api/main.go -o api/docs

# Run security scan
security:
	@echo "Running security scan..."
	cd $(BACKEND_DIR) && gosec ./...
	cd $(FRONTEND_DIR) && npm audit
