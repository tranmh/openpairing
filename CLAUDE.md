# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

OpenPairing is a chess tournament management SaaS platform for the DACH region (Germany, Austria, Switzerland). It features federation integrations (FIDE, DSB, ÖSB, SSB), pairing engines, and real-time chess game support via WebSockets.

## Common Commands

### Development
```bash
make docker-up              # Start PostgreSQL, Redis, MinIO containers
make dev                    # Start both backend API and frontend dev servers
make migrate-up             # Run database migrations
```

### Build
```bash
make build                  # Build all services
make build-backend          # Build Go binaries (api, chess, worker) to bin/
make build-frontend         # Build Next.js app
```

### Testing
```bash
make test                   # Run all tests
make test-backend           # Go tests with race detector and coverage
make test-frontend          # Vitest with coverage

# Run a single Go test
cd backend && go test -v -run TestFunctionName ./internal/package/...

# Run a single frontend test
cd frontend && npm run test -- path/to/file.test.ts
```

### Linting
```bash
make lint                   # Run all linters
make lint-backend           # golangci-lint (5 min timeout)
make lint-frontend          # ESLint, Prettier check, TypeScript check
```

### Formatting
```bash
make fmt                    # Format Go and TypeScript/JavaScript code
```

## Architecture

### Services
- **API** (`backend/cmd/api`): Main REST API server on port 8080
- **Chess Server** (`backend/cmd/chess`): WebSocket server for real-time games on port 8081
- **Worker** (`backend/cmd/worker`): Background job processor
- **Frontend** (`frontend/`): Next.js 14 app with App Router on port 3000

### Backend Structure (`backend/internal/`)
Key packages:
- `auth/` - JWT authentication with access/refresh tokens
- `chess/` - Chess game logic and WebSocket handlers
- `federation/` - FIDE, DSB, ÖSB, SSB API integrations
- `pairing/` - Pairing engine integrations (JaVaFo, bbpPairings, TRF format)
- `tournament/` - Core tournament management
- `player/` - Player profiles and rating management
- `standings/` - Tournament standings calculations
- `export/` - PGN/TRF file exports

Shared utilities in `backend/pkg/`: logger (zerolog), database (pgx), redis, validator, response

### Frontend Structure (`frontend/src/`)
- `app/` - Next.js App Router with route groups: `(auth)`, `(dashboard)`, `play/[gameId]`
- `components/` - React components organized by domain (ui/, chess/, tournament/, player/)
- `store/` - Zustand stores for auth and language state
- `lib/api.ts` - TypeScript API client with automatic token injection
- `i18n/` - Translations (German primary, English)

### Data Flow
```
Frontend (Next.js) → API (Gin) → PostgreSQL
                  ↘ Chess Server (WebSocket) ↔ Redis (pub/sub, game state)
                  ↘ Worker → Federation APIs, Pairing Engines
```

### Key Dependencies
- **Backend**: Gin (HTTP), pgx (PostgreSQL), gorilla/websocket, notnil/chess, zerolog
- **Frontend**: React 18, TanStack Query, Zustand, react-chessboard, react-hook-form + zod, Radix UI

### Infrastructure
- PostgreSQL 16 for persistence
- Redis 7 for caching, sessions, and pub/sub
- MinIO (S3-compatible) for file storage in development

## Database Migrations

Migrations live in `backend/migrations/`. Uses golang-migrate.

```bash
make migrate-up             # Apply migrations
make migrate-down           # Rollback one migration
make migrate-create         # Create new migration (prompts for name)
```

## Environment Configuration

Copy `.env.example` to `.env` for local development. Key variables:
- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_URL` - Redis connection string
- `JWT_SECRET` - JWT signing key
- Ports: API (8080), Chess (8081), Frontend (3000), PostgreSQL (5432), Redis (6379)
