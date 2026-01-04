#!/bin/bash
#
# OpenPairing Deployment Script
# Deploys the complete OpenPairing chess tournament management platform
#
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
PROJECT_NAME="openpairing"
PROJECT_DIR="/home/tranmh/work/openpairing"
INSTALL_DIR="/opt/openpairing"
DATA_DIR="/var/lib/openpairing"
LOG_DIR="/var/log/openpairing"
CONFIG_DIR="/etc/openpairing"

# Service ports (use non-conflicting ports)
API_PORT=8082
CHESS_PORT=8083
FRONTEND_PORT=3001
POSTGRES_PORT=5432
REDIS_PORT=6379

# Database configuration
DB_USER="openpairing"
DB_PASSWORD="openpairing_prod_$(openssl rand -hex 8)"
DB_NAME="openpairing"

# JWT Secret
JWT_SECRET="$(openssl rand -hex 32)"
JWT_REFRESH_SECRET="$(openssl rand -hex 32)"

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

install_system_dependencies() {
    log_info "Installing system dependencies..."

    apt-get update
    apt-get install -y \
        curl \
        wget \
        git \
        build-essential \
        ca-certificates \
        gnupg \
        lsb-release \
        software-properties-common \
        unzip \
        jq

    log_info "System dependencies installed"
}

install_docker() {
    if command -v docker &> /dev/null; then
        log_info "Docker is already installed"
        return
    fi

    log_info "Installing Docker..."

    # Add Docker's official GPG key
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    chmod a+r /etc/apt/keyrings/docker.gpg

    # Add the repository to Apt sources
    echo \
        "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
        $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
        tee /etc/apt/sources.list.d/docker.list > /dev/null

    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

    systemctl enable docker
    systemctl start docker

    log_info "Docker installed successfully"
}

install_golang() {
    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        log_info "Go $GO_VERSION is already installed"
        return
    fi

    log_info "Installing Go 1.22..."

    GO_VERSION="1.22.5"
    wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf /tmp/go.tar.gz
    rm /tmp/go.tar.gz

    # Add to PATH for all users
    if ! grep -q '/usr/local/go/bin' /etc/profile; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    fi
    export PATH=$PATH:/usr/local/go/bin

    log_info "Go installed successfully"
}

install_nodejs() {
    if command -v node &> /dev/null; then
        NODE_VERSION=$(node --version)
        log_info "Node.js $NODE_VERSION is already installed"
        return
    fi

    log_info "Installing Node.js 20 LTS..."

    curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
    apt-get install -y nodejs

    # Install pnpm for faster builds
    npm install -g pnpm

    log_info "Node.js installed successfully"
}

install_migrate() {
    if command -v migrate &> /dev/null; then
        log_info "golang-migrate is already installed"
        return
    fi

    log_info "Installing golang-migrate..."

    curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz -C /usr/local/bin
    chmod +x /usr/local/bin/migrate

    log_info "golang-migrate installed successfully"
}

create_directories() {
    log_info "Creating directories..."

    mkdir -p "$INSTALL_DIR"/{bin,frontend}
    mkdir -p "$DATA_DIR"/{postgres,redis,uploads}
    mkdir -p "$LOG_DIR"
    mkdir -p "$CONFIG_DIR"

    log_info "Directories created"
}

setup_postgres() {
    log_info "Setting up PostgreSQL container..."

    # Check if container exists
    if docker ps -a --format '{{.Names}}' | grep -q "^${PROJECT_NAME}-postgres$"; then
        log_info "PostgreSQL container already exists"
        docker start ${PROJECT_NAME}-postgres 2>/dev/null || true
    else
        docker run -d \
            --name ${PROJECT_NAME}-postgres \
            --restart unless-stopped \
            -e POSTGRES_USER=$DB_USER \
            -e POSTGRES_PASSWORD=$DB_PASSWORD \
            -e POSTGRES_DB=$DB_NAME \
            -v $DATA_DIR/postgres:/var/lib/postgresql/data \
            -p $POSTGRES_PORT:5432 \
            postgres:16-alpine
    fi

    # Wait for PostgreSQL to be ready
    log_info "Waiting for PostgreSQL to be ready..."
    for i in {1..30}; do
        if docker exec ${PROJECT_NAME}-postgres pg_isready -U $DB_USER > /dev/null 2>&1; then
            log_info "PostgreSQL is ready"
            return
        fi
        sleep 1
    done
    log_error "PostgreSQL failed to start"
    exit 1
}

setup_redis() {
    log_info "Setting up Redis container..."

    if docker ps -a --format '{{.Names}}' | grep -q "^${PROJECT_NAME}-redis$"; then
        log_info "Redis container already exists"
        docker start ${PROJECT_NAME}-redis 2>/dev/null || true
    else
        docker run -d \
            --name ${PROJECT_NAME}-redis \
            --restart unless-stopped \
            -v $DATA_DIR/redis:/data \
            -p $REDIS_PORT:6379 \
            redis:7-alpine
    fi

    # Wait for Redis to be ready
    log_info "Waiting for Redis to be ready..."
    for i in {1..30}; do
        if docker exec ${PROJECT_NAME}-redis redis-cli ping > /dev/null 2>&1; then
            log_info "Redis is ready"
            return
        fi
        sleep 1
    done
    log_error "Redis failed to start"
    exit 1
}

run_migrations() {
    log_info "Running database migrations..."

    export PATH=$PATH:/usr/local/go/bin

    # Wait a bit more for postgres to fully initialize
    sleep 3

    cd "$PROJECT_DIR/backend"
    migrate -path ./migrations \
        -database "postgres://$DB_USER:$DB_PASSWORD@localhost:$POSTGRES_PORT/$DB_NAME?sslmode=disable" \
        up

    log_info "Migrations completed"
}

build_backend() {
    log_info "Building backend services..."

    export PATH=$PATH:/usr/local/go/bin
    cd "$PROJECT_DIR/backend"

    # Download dependencies
    go mod download

    # Build API
    CGO_ENABLED=0 go build -ldflags="-w -s" -o "$INSTALL_DIR/bin/api" ./cmd/api
    log_info "API binary built"

    # Build Chess server
    CGO_ENABLED=0 go build -ldflags="-w -s" -o "$INSTALL_DIR/bin/chess" ./cmd/chess
    log_info "Chess server binary built"

    # Build Worker
    CGO_ENABLED=0 go build -ldflags="-w -s" -o "$INSTALL_DIR/bin/worker" ./cmd/worker
    log_info "Worker binary built"

    log_info "Backend build completed"
}

build_frontend() {
    log_info "Building frontend..."

    cd "$PROJECT_DIR/frontend"

    # Install dependencies
    npm ci || npm install

    # Build for production
    NEXT_PUBLIC_API_URL="http://localhost:$API_PORT" \
    NEXT_PUBLIC_WS_URL="ws://localhost:$CHESS_PORT" \
    npm run build

    # Copy build artifacts for standalone mode
    cp -r .next "$INSTALL_DIR/frontend/"

    # For standalone mode, copy static files into standalone directory
    if [ -d ".next/standalone" ]; then
        cp -r .next/static "$INSTALL_DIR/frontend/.next/standalone/.next/" 2>/dev/null || true
        cp -r public "$INSTALL_DIR/frontend/.next/standalone/" 2>/dev/null || true
    fi

    cp -r public "$INSTALL_DIR/frontend/" 2>/dev/null || true
    cp -r node_modules "$INSTALL_DIR/frontend/"
    cp package.json "$INSTALL_DIR/frontend/"
    cp next.config.js "$INSTALL_DIR/frontend/" 2>/dev/null || true

    log_info "Frontend build completed"
}

create_env_file() {
    log_info "Creating environment configuration from .env.example..."

    # Copy .env.example as base
    cp "$PROJECT_DIR/.env.example" "$INSTALL_DIR/.env"

    # Replace values with production settings using sed
    sed -i "s|^POSTGRES_USER=.*|POSTGRES_USER=$DB_USER|" "$INSTALL_DIR/.env"
    sed -i "s|^POSTGRES_PASSWORD=.*|POSTGRES_PASSWORD=$DB_PASSWORD|" "$INSTALL_DIR/.env"
    sed -i "s|^POSTGRES_DB=.*|POSTGRES_DB=$DB_NAME|" "$INSTALL_DIR/.env"
    sed -i "s|^POSTGRES_PORT=.*|POSTGRES_PORT=$POSTGRES_PORT|" "$INSTALL_DIR/.env"
    sed -i "s|^DATABASE_URL=.*|DATABASE_URL=postgres://$DB_USER:$DB_PASSWORD@localhost:$POSTGRES_PORT/$DB_NAME?sslmode=disable|" "$INSTALL_DIR/.env"

    sed -i "s|^REDIS_PORT=.*|REDIS_PORT=$REDIS_PORT|" "$INSTALL_DIR/.env"
    sed -i "s|^REDIS_URL=.*|REDIS_URL=redis://localhost:$REDIS_PORT|" "$INSTALL_DIR/.env"

    sed -i "s|^API_PORT=.*|API_PORT=$API_PORT|" "$INSTALL_DIR/.env"
    sed -i "s|^CHESS_PORT=.*|CHESS_PORT=$CHESS_PORT|" "$INSTALL_DIR/.env"
    sed -i "s|^FRONTEND_PORT=.*|FRONTEND_PORT=$FRONTEND_PORT|" "$INSTALL_DIR/.env"

    sed -i "s|^JWT_SECRET=.*|JWT_SECRET=$JWT_SECRET|" "$INSTALL_DIR/.env"
    sed -i "s|^ENVIRONMENT=.*|ENVIRONMENT=production|" "$INSTALL_DIR/.env"
    sed -i "s|^LOG_LEVEL=.*|LOG_LEVEL=info|" "$INSTALL_DIR/.env"

    sed -i "s|^NEXT_PUBLIC_API_URL=.*|NEXT_PUBLIC_API_URL=http://localhost:$API_PORT|" "$INSTALL_DIR/.env"
    sed -i "s|^NEXT_PUBLIC_WS_URL=.*|NEXT_PUBLIC_WS_URL=ws://localhost:$CHESS_PORT|" "$INSTALL_DIR/.env"
    sed -i "s|^NEXT_PUBLIC_APP_URL=.*|NEXT_PUBLIC_APP_URL=http://localhost:$FRONTEND_PORT|" "$INSTALL_DIR/.env"
    sed -i "s|^CORS_ALLOWED_ORIGINS=.*|CORS_ALLOWED_ORIGINS=http://localhost:$FRONTEND_PORT|" "$INSTALL_DIR/.env"

    # Add backend-specific variables (different naming convention)
    cat >> "$INSTALL_DIR/.env" << EOF

# ===========================================
# Backend Service Configuration
# (Used by Go services - maps to above values)
# ===========================================
DB_HOST=localhost
DB_PORT=$POSTGRES_PORT
DB_USER=$DB_USER
DB_PASSWORD=$DB_PASSWORD
DB_NAME=$DB_NAME
DB_SSLMODE=disable

REDIS_HOST=localhost

# API server port (used by backend config)
PORT=$API_PORT

# Chess server port (used by chess service)
CHESS_PORT=$CHESS_PORT

JWT_REFRESH_SECRET=$JWT_REFRESH_SECRET

# Federation
FEDERATION_CACHE_DIR=$DATA_DIR/federation
FEDERATION_ENABLE_FIDE=false
FEDERATION_ENABLE_DSB=false
FEDERATION_ENABLE_OESB=false
FEDERATION_ENABLE_SSB=false
EOF

    chmod 600 "$INSTALL_DIR/.env"
    log_info "Environment configuration created at $INSTALL_DIR/.env"
}

install_systemd_services() {
    log_info "Installing systemd services..."

    # Copy service files from repo
    cp "$PROJECT_DIR/deploy/systemd/openpairing-api.service" /etc/systemd/system/
    cp "$PROJECT_DIR/deploy/systemd/openpairing-chess.service" /etc/systemd/system/
    cp "$PROJECT_DIR/deploy/systemd/openpairing-frontend.service" /etc/systemd/system/
    cp "$PROJECT_DIR/deploy/systemd/openpairing-worker.service" /etc/systemd/system/

    # Set permissions
    chmod 644 /etc/systemd/system/openpairing-*.service

    systemctl daemon-reload
    log_info "Systemd services installed"
}

start_services() {
    log_info "Starting services..."

    # Enable and start all services
    systemctl enable openpairing-api openpairing-chess openpairing-worker openpairing-frontend

    systemctl start openpairing-api
    sleep 2
    systemctl start openpairing-chess
    sleep 1
    systemctl start openpairing-worker
    sleep 1
    systemctl start openpairing-frontend

    log_info "All services started"
}

verify_deployment() {
    log_info "Verifying deployment..."

    sleep 5  # Wait for services to fully start

    # Check API health
    if curl -s "http://localhost:$API_PORT/health" | grep -q "healthy"; then
        log_info "✓ API is healthy"
    else
        log_error "✗ API health check failed"
    fi

    # Check services status
    for service in openpairing-api openpairing-chess openpairing-worker openpairing-frontend; do
        if systemctl is-active --quiet $service; then
            log_info "✓ $service is running"
        else
            log_warn "✗ $service is not running"
            systemctl status $service --no-pager || true
        fi
    done

    # Check Docker containers
    for container in ${PROJECT_NAME}-postgres ${PROJECT_NAME}-redis; do
        if docker ps --format '{{.Names}}' | grep -q "^$container$"; then
            log_info "✓ $container is running"
        else
            log_warn "✗ $container is not running"
        fi
    done
}

print_summary() {
    echo ""
    echo "=============================================="
    echo "     OpenPairing Deployment Complete!"
    echo "=============================================="
    echo ""
    echo "Services:"
    echo "  - API:       http://localhost:$API_PORT"
    echo "  - Chess WS:  ws://localhost:$CHESS_PORT"
    echo "  - Frontend:  http://localhost:$FRONTEND_PORT"
    echo "  - PostgreSQL: localhost:$POSTGRES_PORT"
    echo "  - Redis:     localhost:$REDIS_PORT"
    echo ""
    echo "Database credentials:"
    echo "  - User:     $DB_USER"
    echo "  - Password: $DB_PASSWORD"
    echo "  - Database: $DB_NAME"
    echo ""
    echo "Configuration: $CONFIG_DIR/openpairing.env"
    echo "Logs:          $LOG_DIR/"
    echo ""
    echo "Manage services:"
    echo "  systemctl status openpairing-api"
    echo "  systemctl restart openpairing-api"
    echo "  journalctl -u openpairing-api -f"
    echo ""
}

# Main deployment flow
main() {
    echo ""
    echo "=============================================="
    echo "     OpenPairing Deployment Script"
    echo "=============================================="
    echo ""

    check_root
    install_system_dependencies
    install_docker
    install_golang
    install_nodejs
    install_migrate
    create_directories
    setup_postgres
    setup_redis
    run_migrations
    build_backend
    build_frontend
    create_env_file
    install_systemd_services
    start_services
    verify_deployment
    print_summary
}

# Run main
main "$@"
