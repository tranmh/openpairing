# OpenPairing

A modern chess tournament management platform for the DACH region (Germany, Austria, Switzerland).

## Features

- **Tournament Management**: Swiss, Round Robin, Knockout, Team Swiss, Arena formats
- **Pairing Engines**: JaVaFo (FIDE-endorsed) and bbpPairings integration
- **Federation Support**: FIDE, DSB (Germany), ÖSB (Austria), SSB (Switzerland)
- **Real-time Chess**: Play games directly in the browser via WebSocket
- **Standings & Tiebreaks**: 90+ tiebreak options including Buchholz, SB, Direct Encounter
- **Multi-language**: German (primary) and English
- **Export**: TRF, PGN file generation for federation reporting

## Tech Stack

### Backend
- **Go 1.22** with Gin web framework
- **PostgreSQL 16** for persistence
- **Redis 7** for caching, sessions, pub/sub
- **WebSocket** for real-time game play

### Frontend
- **Next.js 14** with App Router
- **React 18** with TypeScript
- **TanStack Query** for data fetching
- **Zustand** for state management
- **Radix UI** + Tailwind CSS for styling

### Pairing Tools
- `tools/javafo.jar` - JaVaFo Swiss pairing engine
- `tools/bbpPairings.exe` - bbpPairings Swiss pairing engine

## Quick Start

### Prerequisites

- Go 1.22+
- Node.js 20+
- Docker & Docker Compose
- Make

### Development Setup

```bash
# Clone the repository
git clone https://github.com/tranmh/openpairing.git
cd openpairing

# Copy environment file
cp .env.example .env

# Start infrastructure (PostgreSQL, Redis, MinIO)
make docker-up

# Run database migrations
make migrate-up

# Seed demo data (optional)
make seed

# Start development servers
make dev
```

The application will be available at:
- Frontend: http://localhost:3000
- API: http://localhost:8080
- Chess WebSocket: ws://localhost:8081

### Default Admin Credentials

After seeding:
- Email: `admin@openpairing.org`
- Password: `admin123`

## Project Structure

```
openpairing/
├── backend/
│   ├── cmd/
│   │   ├── api/          # REST API server
│   │   ├── chess/        # WebSocket server
│   │   ├── worker/       # Background jobs
│   │   └── seed/         # Database seeding
│   ├── internal/
│   │   ├── auth/         # JWT authentication
│   │   ├── chess/        # Game logic & WebSocket
│   │   ├── federation/   # FIDE, DSB, ÖSB, SSB APIs
│   │   ├── pairing/      # Pairing engine integration
│   │   ├── tournament/   # Tournament management
│   │   ├── player/       # Player profiles
│   │   ├── standings/    # Standings calculations
│   │   └── export/       # TRF/PGN exports
│   ├── migrations/       # Database migrations
│   └── pkg/              # Shared utilities
├── frontend/
│   └── src/
│       ├── app/          # Next.js App Router
│       ├── components/   # React components
│       ├── lib/          # API client, utilities
│       ├── store/        # Zustand stores
│       └── i18n/         # Translations
├── tools/                # Pairing engines
├── scripts/              # Deployment scripts
└── deploy/               # Systemd services
```

## Available Commands

### Development

```bash
make dev              # Start all development servers
make docker-up        # Start Docker services
make docker-down      # Stop Docker services
```

### Build

```bash
make build            # Build all services
make build-backend    # Build Go binaries
make build-frontend   # Build Next.js app
```

### Testing

```bash
make test             # Run all tests
make test-backend     # Go tests with coverage
make test-frontend    # Vitest with coverage
```

### Database

```bash
make migrate-up       # Apply migrations
make migrate-down     # Rollback one migration
make migrate-create   # Create new migration
make seed             # Seed demo data
make seed-force       # Clear and reseed
```

### Code Quality

```bash
make lint             # Run all linters
make fmt              # Format code
```

## Configuration

Key environment variables (see `.env.example`):

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | - |
| `REDIS_URL` | Redis connection string | - |
| `JWT_SECRET` | JWT signing key | - |
| `API_PORT` | API server port | 8080 |
| `CHESS_PORT` | WebSocket server port | 8081 |
| `FRONTEND_PORT` | Next.js port | 3000 |
| `JAVAFO_PATH` | Path to javafo.jar | tools/javafo.jar |
| `BBP_PATH` | Path to bbpPairings | tools/bbpPairings.exe |

## Deployment

### Production Deployment

```bash
# Standard deployment (idempotent)
sudo ./scripts/deploy.sh

# Force clean deployment (deletes all data)
sudo ./scripts/deploy.sh --force
```

The deployment script:
1. Installs system dependencies (Docker, Go, Node.js)
2. Sets up PostgreSQL and Redis containers
3. Runs database migrations
4. Seeds demo data
5. Builds and deploys all services
6. Configures systemd services

### Services

After deployment, services are managed via systemd:

```bash
systemctl status openpairing-api
systemctl status openpairing-chess
systemctl status openpairing-worker
systemctl status openpairing-frontend
```

## API Documentation

The API follows REST conventions:

- `POST /api/v1/auth/login` - Authenticate
- `GET /api/v1/tournaments` - List tournaments
- `POST /api/v1/tournaments` - Create tournament
- `GET /api/v1/tournaments/:id/standings` - Get standings
- `POST /api/v1/tournaments/:id/rounds/:round/pair` - Generate pairings

Full API documentation available at `/api/docs` when running in development.

## Tournament Formats

| Format | Description |
|--------|-------------|
| Swiss | Standard Swiss system with Dutch pairing |
| Round Robin | All-play-all format |
| Knockout | Single/double elimination |
| Team Swiss | Team-based Swiss pairing |
| Arena | Continuous pairing (Lichess-style) |

## Federation Integration

OpenPairing integrates with chess federations for:
- Player data import (ratings, titles, FIDE IDs)
- Tournament registration via federation APIs
- Rating report generation (TRF format)

Supported federations:
- **FIDE** - International Chess Federation
- **DSB** - German Chess Federation
- **ÖSB** - Austrian Chess Federation
- **SSB** - Swiss Chess Federation

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes using conventional commits (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Commit Convention

We use [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - New features
- `fix:` - Bug fixes
- `docs:` - Documentation changes
- `refactor:` - Code refactoring
- `test:` - Test additions/changes
- `chore:` - Maintenance tasks

## License

This project is licensed under the GNU Affero General Public License v3.0 (AGPL-3.0). See [agpl-3.0.txt](agpl-3.0.txt) for details.

## Support

For issues and feature requests, please use the [GitHub Issues](https://github.com/tranmh/openpairing/issues) page.
