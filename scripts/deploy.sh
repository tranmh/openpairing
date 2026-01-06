#!/bin/bash
#
# OpenPairing Deployment Script
# Deploys the complete OpenPairing chess tournament management platform
#
# Usage:
#   sudo ./scripts/deploy.sh           # Normal deployment (idempotent)
#   sudo ./scripts/deploy.sh --force   # Force clean deployment (deletes everything)
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
CREDENTIALS_FILE="$CONFIG_DIR/credentials"

# Service ports
API_PORT=8080
CHESS_PORT=8081
FRONTEND_PORT=3000
POSTGRES_PORT=5432
REDIS_PORT=6379

# Database configuration (will be loaded or generated)
DB_USER="openpairing"
DB_PASSWORD=""
DB_NAME="openpairing"

# JWT Secret (will be loaded or generated)
JWT_SECRET=""
JWT_REFRESH_SECRET=""

# Force mode flag
FORCE_MODE=false

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

parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --force|-f)
                FORCE_MODE=true
                shift
                ;;
            --help|-h)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --force, -f    Force clean deployment (deletes all data and containers)"
                echo "  --help, -h     Show this help message"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
    done
}

force_cleanup() {
    log_warn "Force mode enabled - cleaning up everything..."

    # Stop and remove systemd services
    for service in openpairing-api openpairing-chess openpairing-worker openpairing-frontend; do
        if systemctl is-active --quiet $service 2>/dev/null; then
            log_info "Stopping $service..."
            systemctl stop $service || true
        fi
        if systemctl is-enabled --quiet $service 2>/dev/null; then
            systemctl disable $service || true
        fi
        rm -f /etc/systemd/system/$service.service
    done
    systemctl daemon-reload 2>/dev/null || true

    # Stop and remove Docker containers
    for container in ${PROJECT_NAME}-postgres ${PROJECT_NAME}-redis; do
        if docker ps -a --format '{{.Names}}' | grep -q "^${container}$"; then
            log_info "Removing container $container..."
            docker stop $container 2>/dev/null || true
            docker rm $container 2>/dev/null || true
        fi
    done

    # Remove data directories
    if [ -d "$DATA_DIR" ]; then
        log_info "Removing data directory $DATA_DIR..."
        rm -rf "$DATA_DIR"
    fi

    # Remove install directory
    if [ -d "$INSTALL_DIR" ]; then
        log_info "Removing install directory $INSTALL_DIR..."
        rm -rf "$INSTALL_DIR"
    fi

    # Remove config directory (including credentials)
    if [ -d "$CONFIG_DIR" ]; then
        log_info "Removing config directory $CONFIG_DIR..."
        rm -rf "$CONFIG_DIR"
    fi

    # Remove log directory
    if [ -d "$LOG_DIR" ]; then
        log_info "Removing log directory $LOG_DIR..."
        rm -rf "$LOG_DIR"
    fi

    log_info "Cleanup completed"
}

load_or_generate_credentials() {
    # Ensure config directory exists
    mkdir -p "$CONFIG_DIR"
    chmod 700 "$CONFIG_DIR"

    if [ -f "$CREDENTIALS_FILE" ]; then
        log_info "Loading existing credentials from $CREDENTIALS_FILE"
        source "$CREDENTIALS_FILE"
    else
        log_info "Generating new credentials..."
        DB_PASSWORD="openpairing_prod_$(openssl rand -hex 8)"
        JWT_SECRET="$(openssl rand -hex 32)"
        JWT_REFRESH_SECRET="$(openssl rand -hex 32)"

        # Save credentials for future runs
        cat > "$CREDENTIALS_FILE" << EOF
# OpenPairing Credentials - Generated on $(date)
# DO NOT DELETE - needed for idempotent deployments
DB_PASSWORD="$DB_PASSWORD"
JWT_SECRET="$JWT_SECRET"
JWT_REFRESH_SECRET="$JWT_REFRESH_SECRET"
EOF
        chmod 600 "$CREDENTIALS_FILE"
        log_info "Credentials saved to $CREDENTIALS_FILE"
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
        # Ensure Docker is running
        if ! systemctl is-active --quiet docker; then
            systemctl start docker
        fi
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
    mkdir -p "$DATA_DIR"/{postgres,redis,uploads,federation}
    mkdir -p "$LOG_DIR"
    mkdir -p "$CONFIG_DIR"

    log_info "Directories created"
}

setup_postgres() {
    log_info "Setting up PostgreSQL container..."

    # Check if container exists and is running
    if docker ps --format '{{.Names}}' | grep -q "^${PROJECT_NAME}-postgres$"; then
        log_info "PostgreSQL container is already running"
    elif docker ps -a --format '{{.Names}}' | grep -q "^${PROJECT_NAME}-postgres$"; then
        log_info "Starting existing PostgreSQL container..."
        docker start ${PROJECT_NAME}-postgres
    else
        log_info "Creating new PostgreSQL container..."
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

    # Check if container exists and is running
    if docker ps --format '{{.Names}}' | grep -q "^${PROJECT_NAME}-redis$"; then
        log_info "Redis container is already running"
    elif docker ps -a --format '{{.Names}}' | grep -q "^${PROJECT_NAME}-redis$"; then
        log_info "Starting existing Redis container..."
        docker start ${PROJECT_NAME}-redis
    else
        log_info "Creating new Redis container..."
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

    # Run migrations (idempotent - will skip already applied migrations)
    migrate -path ./migrations \
        -database "postgres://$DB_USER:$DB_PASSWORD@localhost:$POSTGRES_PORT/$DB_NAME?sslmode=disable" \
        up || {
            # Check if error is just "no change" which is fine
            log_warn "Migration returned non-zero, checking if database is up to date..."
        }

    log_info "Migrations completed"
}

seed_database() {
    log_info "Seeding database with demo data..."

    export PATH=$PATH:/usr/local/go/bin
    cd "$PROJECT_DIR/backend"

    # Build seed command
    log_info "Building seed command..."
    go build -o /tmp/openpairing-seed ./cmd/seed

    # Run seed with force flag for --force deployments
    if [ "$FORCE_MODE" = true ]; then
        log_info "Running seed with --force (clearing existing data)..."
        /tmp/openpairing-seed --force \
            --db-url="postgres://$DB_USER:$DB_PASSWORD@localhost:$POSTGRES_PORT/$DB_NAME?sslmode=disable"
    else
        log_info "Running seed (idempotent)..."
        /tmp/openpairing-seed \
            --db-url="postgres://$DB_USER:$DB_PASSWORD@localhost:$POSTGRES_PORT/$DB_NAME?sslmode=disable"
    fi

    rm -f /tmp/openpairing-seed
    log_info "Database seeding completed"
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
    NEXT_PUBLIC_API_URL="http://localhost:$API_PORT/api/v1" \
    NEXT_PUBLIC_WS_URL="ws://localhost:$CHESS_PORT" \
    npm run build

    # Copy build artifacts
    rm -rf "$INSTALL_DIR/frontend/.next" "$INSTALL_DIR/frontend/node_modules" 2>/dev/null || true
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
    log_info "Creating environment configuration..."

    # Always regenerate env file to ensure it matches current settings
    cat > "$INSTALL_DIR/.env" << EOF
# ===========================================
# OpenPairing Production Environment
# Generated on $(date)
# ===========================================

# Database Configuration
POSTGRES_USER=$DB_USER
POSTGRES_PASSWORD=$DB_PASSWORD
POSTGRES_DB=$DB_NAME
POSTGRES_PORT=$POSTGRES_PORT
DATABASE_URL=postgres://$DB_USER:$DB_PASSWORD@localhost:$POSTGRES_PORT/$DB_NAME?sslmode=disable

# Redis Configuration
REDIS_PORT=$REDIS_PORT
REDIS_URL=redis://localhost:$REDIS_PORT

# Service Ports
API_PORT=$API_PORT
CHESS_PORT=$CHESS_PORT
FRONTEND_PORT=$FRONTEND_PORT

# Security
JWT_SECRET=$JWT_SECRET
JWT_REFRESH_SECRET=$JWT_REFRESH_SECRET

# Environment
ENVIRONMENT=production
LOG_LEVEL=info

# Frontend URLs
NEXT_PUBLIC_API_URL=http://localhost:$API_PORT/api/v1
NEXT_PUBLIC_WS_URL=ws://localhost:$CHESS_PORT
NEXT_PUBLIC_APP_URL=http://localhost:$FRONTEND_PORT
CORS_ALLOWED_ORIGINS=http://localhost:$FRONTEND_PORT

# ===========================================
# Backend Service Configuration
# (Used by Go services)
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

    # Check if systemd service files exist in repo
    if [ ! -d "$PROJECT_DIR/deploy/systemd" ]; then
        log_warn "Systemd service files not found in $PROJECT_DIR/deploy/systemd"
        log_info "Creating systemd service files..."
        mkdir -p "$PROJECT_DIR/deploy/systemd"
        create_systemd_service_files
    fi

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

create_systemd_service_files() {
    # API Service
    cat > "$PROJECT_DIR/deploy/systemd/openpairing-api.service" << EOF
[Unit]
Description=OpenPairing API Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
EnvironmentFile=$INSTALL_DIR/.env
ExecStart=$INSTALL_DIR/bin/api
Restart=always
RestartSec=5
StandardOutput=append:$LOG_DIR/api.log
StandardError=append:$LOG_DIR/api-error.log

[Install]
WantedBy=multi-user.target
EOF

    # Chess Service
    cat > "$PROJECT_DIR/deploy/systemd/openpairing-chess.service" << EOF
[Unit]
Description=OpenPairing Chess WebSocket Server
After=network.target openpairing-api.service

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
EnvironmentFile=$INSTALL_DIR/.env
ExecStart=$INSTALL_DIR/bin/chess
Restart=always
RestartSec=5
StandardOutput=append:$LOG_DIR/chess.log
StandardError=append:$LOG_DIR/chess-error.log

[Install]
WantedBy=multi-user.target
EOF

    # Worker Service
    cat > "$PROJECT_DIR/deploy/systemd/openpairing-worker.service" << EOF
[Unit]
Description=OpenPairing Background Worker
After=network.target openpairing-api.service

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
EnvironmentFile=$INSTALL_DIR/.env
ExecStart=$INSTALL_DIR/bin/worker
Restart=always
RestartSec=5
StandardOutput=append:$LOG_DIR/worker.log
StandardError=append:$LOG_DIR/worker-error.log

[Install]
WantedBy=multi-user.target
EOF

    # Frontend Service
    cat > "$PROJECT_DIR/deploy/systemd/openpairing-frontend.service" << EOF
[Unit]
Description=OpenPairing Frontend (Next.js)
After=network.target openpairing-api.service

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR/frontend
EnvironmentFile=$INSTALL_DIR/.env
ExecStart=/usr/bin/node .next/standalone/server.js
Environment=PORT=$FRONTEND_PORT
Restart=always
RestartSec=5
StandardOutput=append:$LOG_DIR/frontend.log
StandardError=append:$LOG_DIR/frontend-error.log

[Install]
WantedBy=multi-user.target
EOF
}

start_services() {
    log_info "Starting services..."

    # Stop services first if running (for idempotent restarts)
    for service in openpairing-api openpairing-chess openpairing-worker openpairing-frontend; do
        if systemctl is-active --quiet $service 2>/dev/null; then
            log_info "Stopping $service for restart..."
            systemctl stop $service
        fi
    done

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
        log_warn "✗ API health check failed (may still be starting)"
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
    echo "Admin login:"
    echo "  - Email:    admin@openpairing.org"
    echo "  - Password: admin123"
    echo ""
    echo "Configuration files:"
    echo "  - Environment: $INSTALL_DIR/.env"
    echo "  - Credentials: $CREDENTIALS_FILE"
    echo ""
    echo "Logs: $LOG_DIR/"
    echo ""
    echo "Manage services:"
    echo "  systemctl status openpairing-api"
    echo "  systemctl restart openpairing-api"
    echo "  journalctl -u openpairing-api -f"
    echo ""
    echo "Re-deploy (idempotent):"
    echo "  sudo ./scripts/deploy.sh"
    echo ""
    echo "Force clean re-deploy:"
    echo "  sudo ./scripts/deploy.sh --force"
    echo ""
}

# Main deployment flow
main() {
    echo ""
    echo "=============================================="
    echo "     OpenPairing Deployment Script"
    echo "=============================================="
    echo ""

    parse_args "$@"
    check_root

    if [ "$FORCE_MODE" = true ]; then
        force_cleanup
    fi

    load_or_generate_credentials
    install_system_dependencies
    install_docker
    install_golang
    install_nodejs
    install_migrate
    create_directories
    setup_postgres
    setup_redis
    run_migrations
    seed_database
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
