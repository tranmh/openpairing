# OpenPairing - Chess Tournament SaaS Platform

## Project Overview

**Goal**: Build a modern, cloud-native chess tournament management SaaS platform for the DACH region (Germany, Austria, Switzerland) that competes with Tornelo while offering superior federation integration.

**Key Differentiators**:
- Native DSB/DWZ, ÖSB, and FIDE integration
- Multiple pairing engines (JaVaFo + bbpPairings)
- Custom chess server for online/hybrid events
- AGPL-3.0 open source for transparency
- German-first localization

**Tech Stack**:
- Backend: Go (high performance, excellent concurrency)
- Frontend: React + Next.js (SSR, modern DX)
- Database: PostgreSQL (primary) + Redis (cache/realtime)
- Chess Server: Custom WebSocket-based (Go)
- Pairing Engines: JaVaFo (Java), bbpPairings (C++ via CGO/subprocess)

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           CDN (CloudFlare)                              │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
┌─────────────────────────────────────────────────────────────────────────┐
│                     Next.js Frontend (Vercel/Docker)                    │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐       │
│  │ Tournament  │ │   Player    │ │   Arbiter   │ │    Admin    │       │
│  │   Portal    │ │   Portal    │ │   Console   │ │   Console   │       │
│  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘       │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                          ┌─────────┴─────────┐
                          │   API Gateway     │
                          │   (Kong/Traefik)  │
                          └─────────┬─────────┘
                                    │
        ┌───────────────────────────┼───────────────────────────┐
        │                           │                           │
        ▼                           ▼                           ▼
┌───────────────┐         ┌─────────────────┐         ┌─────────────────┐
│   Core API    │         │  Chess Server   │         │  Worker Service │
│     (Go)      │         │     (Go)        │         │      (Go)       │
│               │         │                 │         │                 │
│ - Auth        │         │ - WebSocket     │         │ - Pairing Jobs  │
│ - Tournaments │         │ - Game State    │         │ - Rating Sync   │
│ - Players     │         │ - Move Valid.   │         │ - Report Gen    │
│ - Pairings    │         │ - Clock Mgmt    │         │ - Notifications │
│ - Results     │         │ - Spectating    │         │ - Fair Play     │
└───────┬───────┘         └────────┬────────┘         └────────┬────────┘
        │                          │                           │
        └──────────────────────────┼───────────────────────────┘
                                   │
                    ┌──────────────┼──────────────┐
                    │              │              │
                    ▼              ▼              ▼
             ┌──────────┐  ┌────────────┐  ┌───────────┐
             │PostgreSQL│  │   Redis    │  │ S3/MinIO  │
             │          │  │            │  │           │
             │- Users   │  │- Sessions  │  │- PGN files│
             │- Tourneys│  │- Game state│  │- Reports  │
             │- Games   │  │- Pub/Sub   │  │- Exports  │
             │- Ratings │  │- Cache     │  │- Backups  │
             └──────────┘  └────────────┘  └───────────┘
                                   │
                    ┌──────────────┼──────────────┐
                    │              │              │
                    ▼              ▼              ▼
             ┌──────────┐  ┌────────────┐  ┌───────────┐
             │  JaVaFo  │  │bbpPairings │  │Federation │
             │  (Java)  │  │   (C++)    │  │   APIs    │
             │          │  │            │  │           │
             │ Pairing  │  │ Pairing    │  │- FIDE XML │
             │ Engine   │  │ Engine     │  │- DSB API  │
             └──────────┘  └────────────┘  │- ÖSB     │
                                           └───────────┘
```

---

## Database Schema

### Core Tables

```sql
-- Users & Authentication
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    role VARCHAR(50) NOT NULL DEFAULT 'player', -- player, arbiter, organizer, admin
    preferred_language VARCHAR(10) DEFAULT 'de',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Player Profiles (linked to users, but can exist independently for imported players)
CREATE TABLE players (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),

    -- Personal Info
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    birth_date DATE,
    gender VARCHAR(10), -- m, f, other

    -- Federation IDs
    fide_id VARCHAR(20) UNIQUE,
    dsb_id VARCHAR(20) UNIQUE,       -- German: PKZ (Personenkennziffer)
    oesb_id VARCHAR(20) UNIQUE,      -- Austrian
    ssb_id VARCHAR(20) UNIQUE,       -- Swiss

    -- Ratings (cached, synced from federations)
    fide_standard INTEGER,
    fide_rapid INTEGER,
    fide_blitz INTEGER,
    dwz INTEGER,                      -- German rating
    dwz_index INTEGER,                -- DWZ development index
    oesb_elo INTEGER,                 -- Austrian ELO

    -- Titles
    fide_title VARCHAR(10),           -- GM, IM, FM, CM, WGM, WIM, WFM, WCM
    national_title VARCHAR(50),

    -- Club Affiliations
    club_id UUID REFERENCES clubs(id),
    federation VARCHAR(10),           -- GER, AUT, SUI, etc.

    -- Metadata
    ratings_updated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_players_fide_id ON players(fide_id);
CREATE INDEX idx_players_dsb_id ON players(dsb_id);
CREATE INDEX idx_players_name ON players(last_name, first_name);

-- Clubs
CREATE TABLE clubs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    short_name VARCHAR(50),
    federation VARCHAR(10),
    dsb_club_id VARCHAR(20),          -- DSB Vereinsnummer
    oesb_club_id VARCHAR(20),
    city VARCHAR(100),
    country VARCHAR(3) DEFAULT 'DEU',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Organizations (tournament organizers, federations)
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50), -- federation, club, private
    logo_url VARCHAR(500),
    website VARCHAR(500),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE organization_members (
    organization_id UUID REFERENCES organizations(id),
    user_id UUID REFERENCES users(id),
    role VARCHAR(50), -- owner, admin, arbiter
    PRIMARY KEY (organization_id, user_id)
);

-- Tournaments
CREATE TABLE tournaments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID REFERENCES organizations(id),

    -- Basic Info
    name VARCHAR(255) NOT NULL,
    description TEXT,
    slug VARCHAR(100) UNIQUE,

    -- Type & Format
    type VARCHAR(50) NOT NULL,        -- swiss, round_robin, knockout, team_swiss
    mode VARCHAR(50) NOT NULL,        -- otb, online, hybrid
    time_control VARCHAR(100),        -- e.g., "90+30" or "15+10"
    time_control_type VARCHAR(20),    -- standard, rapid, blitz

    -- Pairing Configuration
    pairing_engine VARCHAR(50) DEFAULT 'javafo', -- javafo, bbp
    pairing_system VARCHAR(50) DEFAULT 'dutch',  -- dutch, burstein, dubov
    accelerated_pairing BOOLEAN DEFAULT FALSE,

    -- Dates
    start_date DATE NOT NULL,
    end_date DATE,
    registration_opens TIMESTAMPTZ,
    registration_closes TIMESTAMPTZ,

    -- Limits
    max_players INTEGER,
    min_players INTEGER DEFAULT 4,
    rounds INTEGER NOT NULL,

    -- Location (for OTB/hybrid)
    venue_name VARCHAR(255),
    venue_address TEXT,
    venue_city VARCHAR(100),
    venue_country VARCHAR(3),

    -- Rating & Reporting
    fide_rated BOOLEAN DEFAULT FALSE,
    dwz_rated BOOLEAN DEFAULT FALSE,
    oesb_rated BOOLEAN DEFAULT FALSE,

    -- Fees
    entry_fee_cents INTEGER DEFAULT 0,
    currency VARCHAR(3) DEFAULT 'EUR',

    -- Status
    status VARCHAR(50) DEFAULT 'draft', -- draft, published, registration, ongoing, completed, cancelled

    -- Tiebreaks (JSON array of tiebreak codes in order)
    tiebreaks JSONB DEFAULT '["buchholz_cut1", "buchholz", "sonneborn_berger"]',

    -- Settings
    settings JSONB DEFAULT '{}',

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_tournaments_status ON tournaments(status);
CREATE INDEX idx_tournaments_dates ON tournaments(start_date, end_date);

-- Tournament Sections (for multi-section events)
CREATE TABLE tournament_sections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tournament_id UUID REFERENCES tournaments(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    rating_min INTEGER,
    rating_max INTEGER,
    age_min INTEGER,
    age_max INTEGER,
    gender VARCHAR(10),
    sort_order INTEGER DEFAULT 0
);

-- Tournament Participants
CREATE TABLE tournament_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tournament_id UUID REFERENCES tournaments(id) ON DELETE CASCADE,
    section_id UUID REFERENCES tournament_sections(id),
    player_id UUID REFERENCES players(id),

    -- Registration
    registration_status VARCHAR(50) DEFAULT 'registered', -- registered, confirmed, withdrawn, expelled
    registered_at TIMESTAMPTZ DEFAULT NOW(),

    -- Starting Info
    start_rank INTEGER,
    starting_rating INTEGER,          -- Rating used for this tournament

    -- Results (denormalized for performance)
    points DECIMAL(4,1) DEFAULT 0,
    buchholz DECIMAL(6,2) DEFAULT 0,
    buchholz_cut1 DECIMAL(6,2) DEFAULT 0,
    sonneborn_berger DECIMAL(6,2) DEFAULT 0,
    wins INTEGER DEFAULT 0,
    draws INTEGER DEFAULT 0,
    losses INTEGER DEFAULT 0,

    -- Status
    is_active BOOLEAN DEFAULT TRUE,   -- FALSE if withdrawn
    bye_requests JSONB DEFAULT '[]',  -- Rounds where bye requested

    -- Payment
    payment_status VARCHAR(50) DEFAULT 'pending',
    payment_reference VARCHAR(100),

    UNIQUE(tournament_id, player_id)
);

CREATE INDEX idx_participants_tournament ON tournament_participants(tournament_id);
CREATE INDEX idx_participants_points ON tournament_participants(tournament_id, points DESC);

-- Tournament Rounds
CREATE TABLE tournament_rounds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tournament_id UUID REFERENCES tournaments(id) ON DELETE CASCADE,
    round_number INTEGER NOT NULL,

    -- Timing
    scheduled_start TIMESTAMPTZ,
    actual_start TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    -- Status
    status VARCHAR(50) DEFAULT 'pending', -- pending, pairing, paired, ongoing, completed

    -- Pairing metadata
    paired_at TIMESTAMPTZ,
    paired_by UUID REFERENCES users(id),
    pairing_engine_used VARCHAR(50),

    UNIQUE(tournament_id, round_number)
);

-- Games/Pairings
CREATE TABLE games (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tournament_id UUID REFERENCES tournaments(id) ON DELETE CASCADE,
    round_id UUID REFERENCES tournament_rounds(id) ON DELETE CASCADE,
    section_id UUID REFERENCES tournament_sections(id),

    -- Pairing
    board_number INTEGER NOT NULL,
    white_participant_id UUID REFERENCES tournament_participants(id),
    black_participant_id UUID REFERENCES tournament_participants(id),

    -- Result
    result VARCHAR(10),               -- 1-0, 0-1, 1/2-1/2, 1-0F, 0-1F, 0-0F, BYE
    white_score DECIMAL(3,1),
    black_score DECIMAL(3,1),

    -- For online games
    online_game_id UUID,              -- Reference to chess_games table

    -- Status
    status VARCHAR(50) DEFAULT 'scheduled', -- scheduled, ongoing, completed, adjourned

    -- Metadata
    result_entered_by UUID REFERENCES users(id),
    result_entered_at TIMESTAMPTZ,
    pgn TEXT,

    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_games_tournament_round ON games(tournament_id, round_id);
CREATE INDEX idx_games_participants ON games(white_participant_id, black_participant_id);

-- Forbidden Pairs (players who shouldn't be paired)
CREATE TABLE forbidden_pairs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tournament_id UUID REFERENCES tournaments(id) ON DELETE CASCADE,
    player1_id UUID REFERENCES tournament_participants(id),
    player2_id UUID REFERENCES tournament_participants(id),
    reason VARCHAR(255),              -- same_club, siblings, requested
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Online Chess Games (for the custom chess server)
CREATE TABLE chess_games (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Players
    white_user_id UUID REFERENCES users(id),
    black_user_id UUID REFERENCES users(id),

    -- Game State
    fen VARCHAR(100) DEFAULT 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1',
    moves TEXT,                       -- UCI format: e2e4 e7e5 g1f3...
    move_count INTEGER DEFAULT 0,

    -- Clocks (milliseconds remaining)
    white_clock_ms INTEGER,
    black_clock_ms INTEGER,
    initial_time_ms INTEGER,
    increment_ms INTEGER,

    -- Status
    status VARCHAR(50) DEFAULT 'waiting', -- waiting, ongoing, completed, aborted
    result VARCHAR(10),
    termination VARCHAR(50),          -- checkmate, resignation, timeout, draw_agreement, stalemate

    -- Timestamps
    started_at TIMESTAMPTZ,
    last_move_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_chess_games_status ON chess_games(status);
CREATE INDEX idx_chess_games_players ON chess_games(white_user_id, black_user_id);

-- Rating History (track all rating changes)
CREATE TABLE rating_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id UUID REFERENCES players(id),
    rating_type VARCHAR(20),          -- fide_standard, fide_rapid, dwz, oesb
    old_rating INTEGER,
    new_rating INTEGER,
    tournament_id UUID REFERENCES tournaments(id),
    change_date DATE,
    source VARCHAR(50),               -- sync, tournament_result
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Audit Log
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50),
    entity_id UUID,
    old_value JSONB,
    new_value JSONB,
    ip_address INET,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_audit_log_entity ON audit_log(entity_type, entity_id);

-- Notifications
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    type VARCHAR(50),                 -- pairing_published, game_starting, result_entered
    title VARCHAR(255),
    message TEXT,
    data JSONB,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Federation Sync Log
CREATE TABLE federation_sync_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    federation VARCHAR(20),           -- fide, dsb, oesb
    sync_type VARCHAR(50),            -- full, incremental
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    players_updated INTEGER,
    status VARCHAR(50),
    error_message TEXT
);

-- ==========================================
-- TEAM TOURNAMENT TABLES
-- ==========================================

-- Teams
CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    club_id UUID REFERENCES clubs(id),
    name VARCHAR(255) NOT NULL,
    short_name VARCHAR(50),

    -- For league play (Bundesliga, etc.)
    division VARCHAR(100),            -- 1. Bundesliga, 2. Bundesliga, etc.
    region VARCHAR(100),              -- For regional leagues

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Team Members (players assigned to a team)
CREATE TABLE team_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID REFERENCES teams(id) ON DELETE CASCADE,
    player_id UUID REFERENCES players(id),

    board_number INTEGER,             -- Default board assignment (1-8, etc.)
    is_captain BOOLEAN DEFAULT FALSE,
    is_reserve BOOLEAN DEFAULT FALSE,

    -- Season/validity
    valid_from DATE,
    valid_until DATE,

    created_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(team_id, player_id)
);

CREATE INDEX idx_team_members_team ON team_members(team_id);
CREATE INDEX idx_team_members_player ON team_members(player_id);

-- Team Tournament Participants (teams registered for a tournament)
CREATE TABLE team_tournament_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tournament_id UUID REFERENCES tournaments(id) ON DELETE CASCADE,
    team_id UUID REFERENCES teams(id),

    -- Registration
    registration_status VARCHAR(50) DEFAULT 'registered',
    registered_at TIMESTAMPTZ DEFAULT NOW(),

    -- Starting info
    start_rank INTEGER,
    seed_rating INTEGER,              -- Average or sum of board ratings

    -- Results (denormalized)
    match_points DECIMAL(4,1) DEFAULT 0,  -- 2 for win, 1 for draw, 0 for loss
    board_points DECIMAL(5,1) DEFAULT 0,  -- Sum of individual game points
    buchholz DECIMAL(6,2) DEFAULT 0,

    -- Lineup submitted for current round
    lineup_locked BOOLEAN DEFAULT FALSE,

    UNIQUE(tournament_id, team_id)
);

CREATE INDEX idx_team_participants_tournament ON team_tournament_participants(tournament_id);

-- Team Lineups (per round)
CREATE TABLE team_lineups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tournament_id UUID REFERENCES tournaments(id) ON DELETE CASCADE,
    round_id UUID REFERENCES tournament_rounds(id) ON DELETE CASCADE,
    team_participant_id UUID REFERENCES team_tournament_participants(id) ON DELETE CASCADE,

    -- Lineup details (JSON array of player assignments)
    -- Format: [{"board": 1, "player_id": "uuid", "rating": 2450}, ...]
    lineup JSONB NOT NULL,

    -- Validation
    is_valid BOOLEAN DEFAULT TRUE,
    validation_errors JSONB,          -- Any lineup rule violations

    submitted_at TIMESTAMPTZ,
    submitted_by UUID REFERENCES users(id),

    UNIQUE(round_id, team_participant_id)
);

-- Team Matches (pairing between two teams)
CREATE TABLE team_matches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tournament_id UUID REFERENCES tournaments(id) ON DELETE CASCADE,
    round_id UUID REFERENCES tournament_rounds(id) ON DELETE CASCADE,

    match_number INTEGER NOT NULL,

    -- Teams
    home_team_id UUID REFERENCES team_tournament_participants(id),
    away_team_id UUID REFERENCES team_tournament_participants(id),

    -- Result
    home_match_points DECIMAL(3,1),   -- 2, 1, or 0
    away_match_points DECIMAL(3,1),
    home_board_points DECIMAL(4,1),   -- Sum of board results (e.g., 4.5)
    away_board_points DECIMAL(4,1),

    status VARCHAR(50) DEFAULT 'scheduled',

    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_team_matches_round ON team_matches(tournament_id, round_id);

-- Team Match Boards (individual games within a team match)
CREATE TABLE team_match_boards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_match_id UUID REFERENCES team_matches(id) ON DELETE CASCADE,
    game_id UUID REFERENCES games(id), -- Links to the individual game

    board_number INTEGER NOT NULL,

    -- Players (from lineup)
    home_player_id UUID REFERENCES players(id),
    away_player_id UUID REFERENCES players(id),
    home_rating INTEGER,
    away_rating INTEGER,

    -- Result
    home_score DECIMAL(3,1),
    away_score DECIMAL(3,1),

    UNIQUE(team_match_id, board_number)
);

CREATE INDEX idx_team_match_boards ON team_match_boards(team_match_id);

-- Team Tournament Settings (additional settings for team events)
CREATE TABLE team_tournament_settings (
    tournament_id UUID PRIMARY KEY REFERENCES tournaments(id) ON DELETE CASCADE,

    boards_per_match INTEGER NOT NULL DEFAULT 8,

    -- Lineup rules
    lineup_submission_deadline_minutes INTEGER DEFAULT 30, -- Before round start
    allow_late_lineup BOOLEAN DEFAULT FALSE,

    -- Rating order rules
    enforce_rating_order BOOLEAN DEFAULT TRUE,  -- Higher rated on lower board
    rating_order_tolerance INTEGER DEFAULT 0,   -- Allowed ELO difference for swap

    -- Scoring
    match_win_points DECIMAL(3,1) DEFAULT 2,
    match_draw_points DECIMAL(3,1) DEFAULT 1,
    match_loss_points DECIMAL(3,1) DEFAULT 0,

    -- Tiebreaks for team standings
    team_tiebreaks JSONB DEFAULT '["board_points", "buchholz", "direct_encounter"]',

    -- Board point rules
    board_win_points DECIMAL(3,1) DEFAULT 1,
    board_draw_points DECIMAL(3,1) DEFAULT 0.5,
    board_loss_points DECIMAL(3,1) DEFAULT 0,

    -- Default result for unplayed boards
    default_board_result VARCHAR(20) DEFAULT 'loss' -- 'loss', 'draw', 'none'
);
```

---

## API Design

### REST API Endpoints

```
Authentication
├── POST   /api/v1/auth/register
├── POST   /api/v1/auth/login
├── POST   /api/v1/auth/logout
├── POST   /api/v1/auth/refresh
├── POST   /api/v1/auth/forgot-password
└── POST   /api/v1/auth/reset-password

Players
├── GET    /api/v1/players                    # Search players
├── GET    /api/v1/players/:id                # Get player details
├── POST   /api/v1/players                    # Create player (admin)
├── PUT    /api/v1/players/:id                # Update player
├── GET    /api/v1/players/:id/history        # Rating history
├── POST   /api/v1/players/import/fide        # Import from FIDE
├── POST   /api/v1/players/import/dsb         # Import from DSB
└── POST   /api/v1/players/search-federation  # Search federation DBs

Tournaments
├── GET    /api/v1/tournaments                # List tournaments
├── POST   /api/v1/tournaments                # Create tournament
├── GET    /api/v1/tournaments/:id            # Get tournament details
├── PUT    /api/v1/tournaments/:id            # Update tournament
├── DELETE /api/v1/tournaments/:id            # Delete tournament
├── POST   /api/v1/tournaments/:id/publish    # Publish tournament
├── GET    /api/v1/tournaments/:id/standings  # Get standings
├── GET    /api/v1/tournaments/:id/crosstable # Get crosstable
└── GET    /api/v1/tournaments/:id/export/:format # Export (trf, pgn, pdf)

Tournament Participants
├── GET    /api/v1/tournaments/:id/participants
├── POST   /api/v1/tournaments/:id/participants          # Register
├── DELETE /api/v1/tournaments/:id/participants/:pid     # Withdraw
├── PUT    /api/v1/tournaments/:id/participants/:pid     # Update
└── POST   /api/v1/tournaments/:id/participants/import   # Bulk import

Rounds & Pairings
├── GET    /api/v1/tournaments/:id/rounds
├── POST   /api/v1/tournaments/:id/rounds/:num/pair      # Generate pairings
├── GET    /api/v1/tournaments/:id/rounds/:num/pairings  # Get pairings
├── PUT    /api/v1/tournaments/:id/rounds/:num/pairings  # Manual edit
├── POST   /api/v1/tournaments/:id/rounds/:num/start     # Start round
└── POST   /api/v1/tournaments/:id/rounds/:num/complete  # Complete round

Results
├── GET    /api/v1/tournaments/:id/games
├── GET    /api/v1/tournaments/:id/games/:gid
├── PUT    /api/v1/tournaments/:id/games/:gid/result     # Enter result
└── GET    /api/v1/tournaments/:id/games/:gid/pgn        # Get PGN

Reports & Export
├── GET    /api/v1/tournaments/:id/reports/trf           # FIDE TRF
├── GET    /api/v1/tournaments/:id/reports/dwz           # DWZ submission
├── GET    /api/v1/tournaments/:id/reports/norms         # Title norms
├── POST   /api/v1/tournaments/:id/submit/fide           # Submit to FIDE
└── POST   /api/v1/tournaments/:id/submit/dsb            # Submit to DSB

Organizations
├── GET    /api/v1/organizations
├── POST   /api/v1/organizations
├── GET    /api/v1/organizations/:id
├── PUT    /api/v1/organizations/:id
└── GET    /api/v1/organizations/:id/tournaments

Online Games (Chess Server)
├── GET    /api/v1/games/:id                  # Get game state
├── POST   /api/v1/games/:id/resign
├── POST   /api/v1/games/:id/draw-offer
├── POST   /api/v1/games/:id/abort
└── POST   /api/v1/games/:id/call-arbiter

Teams & Team Tournaments
├── GET    /api/v1/teams                              # List teams
├── POST   /api/v1/teams                              # Create team
├── GET    /api/v1/teams/:id                          # Get team details
├── PUT    /api/v1/teams/:id                          # Update team
├── GET    /api/v1/teams/:id/members                  # Get team roster
├── POST   /api/v1/teams/:id/members                  # Add player to team
├── DELETE /api/v1/teams/:id/members/:pid             # Remove player from team
├── GET    /api/v1/clubs/:id/teams                    # List club's teams

Team Tournament Registration
├── GET    /api/v1/tournaments/:id/teams              # List registered teams
├── POST   /api/v1/tournaments/:id/teams              # Register team
├── DELETE /api/v1/tournaments/:id/teams/:tid         # Withdraw team
├── GET    /api/v1/tournaments/:id/teams/:tid/lineup  # Get current lineup

Team Lineups
├── GET    /api/v1/tournaments/:id/rounds/:r/lineups           # All lineups for round
├── GET    /api/v1/tournaments/:id/rounds/:r/lineups/:tid      # Team's lineup
├── PUT    /api/v1/tournaments/:id/rounds/:r/lineups/:tid      # Submit/update lineup
├── POST   /api/v1/tournaments/:id/rounds/:r/lineups/:tid/lock # Lock lineup

Team Matches
├── GET    /api/v1/tournaments/:id/matches                     # All team matches
├── GET    /api/v1/tournaments/:id/rounds/:r/matches           # Matches for round
├── GET    /api/v1/tournaments/:id/matches/:mid                # Match details
├── GET    /api/v1/tournaments/:id/matches/:mid/boards         # Board pairings
├── PUT    /api/v1/tournaments/:id/matches/:mid/boards/:b      # Enter board result
├── GET    /api/v1/tournaments/:id/team-standings              # Team standings
```

### WebSocket Events (Chess Server)

```
Client → Server:
├── join_game { game_id }
├── leave_game { game_id }
├── make_move { game_id, move: "e2e4" }
├── offer_draw { game_id }
├── accept_draw { game_id }
├── resign { game_id }
├── call_arbiter { game_id, message }
└── spectate { game_id }

Server → Client:
├── game_state { fen, moves, clocks, status }
├── move_made { move, fen, clocks }
├── game_over { result, termination }
├── draw_offered { by }
├── draw_declined
├── clock_update { white_ms, black_ms }
├── arbiter_message { message }
├── opponent_connected
├── opponent_disconnected
└── spectator_count { count }
```

---

## Feature Modules

### Module 1: Authentication & User Management

**Features:**
- Email/password registration and login
- OAuth2 (Google, optional)
- JWT-based authentication with refresh tokens
- Role-based access control (player, arbiter, organizer, admin)
- Email verification
- Password reset flow
- User profile management
- Preferred language setting

**Files to create:**
```
backend/
├── internal/auth/
│   ├── handler.go          # HTTP handlers
│   ├── service.go          # Business logic
│   ├── jwt.go              # Token management
│   └── middleware.go       # Auth middleware
├── internal/user/
│   ├── handler.go
│   ├── service.go
│   └── repository.go
```

### Module 2: Player Database & Federation Sync

**Features:**
- Unified player profiles (FIDE + DWZ + ÖSB)
- FIDE XML import (monthly full sync)
- DSB API integration (weekly incremental)
- ÖSB data sync
- Player search across all federations
- Rating conflict detection
- Rating history tracking
- Player merge (deduplication)

**Federation Integration Details:**

```
FIDE:
├── Download: https://ratings.fide.com/download/players_list_xml.zip
├── Format: XML
├── Fields: fide_id, name, federation, sex, title, rating_std, rating_rpd, rating_blz
└── Frequency: Monthly

DSB (DeWIS):
├── API: https://www.schachbund.de/php/dewis/
├── Auth: API key (register at schachbund.de)
├── Endpoints:
│   ├── spieler.php?pkz={id}&format=csv
│   └── verein.php?zps={club_id}&format=csv
├── Fields: pkz, name, dwz, index, verein, status
└── Frequency: Weekly updates

ÖSB:
├── Source: chess-results.com / eloliste.chess.at
├── Format: HTML scraping or CSV export
├── Fields: oesb_id, name, elo, club
└── Frequency: Monthly
```

**Files to create:**
```
backend/
├── internal/player/
│   ├── handler.go
│   ├── service.go
│   ├── repository.go
│   └── search.go
├── internal/federation/
│   ├── fide/
│   │   ├── importer.go     # XML parser
│   │   └── scheduler.go    # Monthly sync job
│   ├── dsb/
│   │   ├── client.go       # API client
│   │   ├── importer.go
│   │   └── scheduler.go
│   └── oesb/
│       ├── scraper.go
│       └── importer.go
```

### Module 3: Tournament Management

**Features:**
- Tournament CRUD
- Multiple formats: Swiss, Round Robin, Knockout, Team
- Multi-section support (rating groups, age groups)
- Configurable tiebreaks (90+ options)
- Registration management
- Bulk player import (CSV)
- Waitlist management
- Bye request handling
- Payment processing (Stripe + SEPA)
- Tournament cloning

**Tiebreak Systems (implement all):**
```go
const (
    TBBuchholz        = "buchholz"
    TBBuchholzCut1    = "buchholz_cut1"
    TBBuchholzCut2    = "buchholz_cut2"
    TBBuchholzSum     = "buchholz_sum"
    TBSonnebornBerger = "sonneborn_berger"
    TBProgress        = "progress"
    TBWins            = "wins"
    TBGamesBlack      = "games_black"
    TBDirectEncounter = "direct_encounter"
    TBAROC            = "aroc"              // Average Rating of Opponents Cut
    TBKoya            = "koya"              // % against players with 50%+
    // ... 80+ more
)
```

**Files to create:**
```
backend/
├── internal/tournament/
│   ├── handler.go
│   ├── service.go
│   ├── repository.go
│   ├── registration.go
│   ├── tiebreak/
│   │   ├── calculator.go
│   │   ├── buchholz.go
│   │   ├── sonneborn.go
│   │   └── ... (each tiebreak)
│   └── export/
│       ├── crosstable.go
│       ├── standings.go
│       └── pdf.go
```

### Module 4: Pairing Engine Integration

**Features:**
- JaVaFo integration (primary, FIDE-endorsed)
- bbpPairings integration (secondary, C++)
- TRF(x) format support (input/output)
- Manual pairing override
- Pairing preview before publishing
- Pairing validation
- Color preference handling
- Accelerated pairing modes

**JaVaFo Integration:**
```go
// Run JaVaFo as subprocess
// Input: TRF file
// Output: Pairings in TRF format

func (p *JaVaFoPairer) GeneratePairings(tournament *Tournament, round int) (*Pairings, error) {
    // 1. Generate TRF file from tournament state
    trf := p.generateTRF(tournament, round)

    // 2. Write to temp file
    tmpFile := writeTempFile(trf)

    // 3. Execute JaVaFo
    cmd := exec.Command("java", "-jar", "javafo.jar", tmpFile)
    output, err := cmd.Output()

    // 4. Parse output pairings
    return p.parsePairings(output)
}
```

**bbpPairings Integration:**
```go
// Compile bbpPairings or use pre-built binary
// TRF(bx) format (extended TRF)

func (p *BBPPairer) GeneratePairings(tournament *Tournament, round int) (*Pairings, error) {
    trf := p.generateTRFbx(tournament, round)
    cmd := exec.Command("./bbpPairings", "--dutch", "-p", tmpFile)
    // ...
}
```

**Files to create:**
```
backend/
├── internal/pairing/
│   ├── handler.go
│   ├── service.go
│   ├── engine/
│   │   ├── interface.go    # Pairing engine interface
│   │   ├── javafo.go       # JaVaFo wrapper
│   │   ├── bbp.go          # bbpPairings wrapper
│   │   └── manual.go       # Manual pairing
│   ├── trf/
│   │   ├── parser.go       # TRF parser
│   │   ├── generator.go    # TRF generator
│   │   └── validator.go    # TRF validation
│   └── validator.go        # Pairing validation
├── bin/
│   ├── javafo.jar
│   └── bbpPairings
```

### Module 5: Results & Standings

**Features:**
- Result entry (by arbiter or players)
- Result confirmation flow (for online games)
- Automatic tiebreak recalculation
- Real-time standings update
- Crosstable generation
- Performance rating calculation
- Title norm checking
- Rating change preview

**Files to create:**
```
backend/
├── internal/result/
│   ├── handler.go
│   ├── service.go
│   └── validator.go
├── internal/standings/
│   ├── calculator.go
│   ├── crosstable.go
│   └── norms.go
```

### Module 6: Chess Server (Online Play)

**Features:**
- WebSocket-based real-time games
- Move validation (use chess library)
- Clock management
- Draw offers/accepts
- Resignation
- Abort (before move 2)
- Spectating
- Game reconnection
- PGN generation
- Move broadcast (for spectators)

**Architecture:**
```
┌─────────────────────────────────────────────────────────────┐
│                    WebSocket Gateway                        │
│                      (Go + Gorilla)                         │
└─────────────────────────────────────────────────────────────┘
                            │
            ┌───────────────┼───────────────┐
            ▼               ▼               ▼
    ┌───────────────┐ ┌───────────┐ ┌───────────────┐
    │ Game Manager  │ │  Clock    │ │  Spectator    │
    │               │ │  Service  │ │  Manager      │
    │ - State mgmt  │ │           │ │               │
    │ - Move valid. │ │ - Timers  │ │ - Pub/Sub     │
    │ - Game logic  │ │ - Updates │ │ - Broadcast   │
    └───────────────┘ └───────────┘ └───────────────┘
            │               │               │
            └───────────────┼───────────────┘
                            ▼
                    ┌───────────────┐
                    │     Redis     │
                    │               │
                    │ - Game state  │
                    │ - Pub/Sub     │
                    │ - Clock sync  │
                    └───────────────┘
```

**Files to create:**
```
backend/
├── internal/chess/
│   ├── game.go             # Game state management
│   ├── board.go            # Board representation
│   ├── move.go             # Move validation
│   ├── clock.go            # Clock management
│   └── pgn.go              # PGN generation
├── internal/server/
│   ├── websocket.go        # WS connection handling
│   ├── hub.go              # Connection hub
│   ├── game_handler.go     # Game message handlers
│   └── spectator.go        # Spectator management
```

### Module 7: Arbiter Tools

**Features:**
- Arbiter console (dedicated UI)
- Manual result entry
- Manual pairing adjustment
- Player withdrawal mid-tournament
- Bye assignment (full/half point)
- Clock adjustment (for online)
- Game pause/resume
- Player expulsion
- Appeal management
- Real-time tournament monitor
- Multi-arbiter support
- Arbiter chat

**Files to create:**
```
backend/
├── internal/arbiter/
│   ├── handler.go
│   ├── service.go
│   ├── console.go          # Real-time dashboard data
│   └── actions.go          # Arbiter-specific actions
```

### Module 8: Reporting & Federation Submission

**Features:**
- FIDE TRF16 export
- DWZ submission format
- ÖSB export
- Title norm reports
- Tournament certificates (PDF)
- Crosstable PDF/HTML
- Standings PDF/HTML
- Direct submission to DSB API
- Chess-Results.com integration

**TRF16 Format:**
```
012 Tournament Name
022 City
032 Federation
042 Start Date
052 End Date
062 Rounds
072 Type (Swiss)
082 Chief Arbiter
092 Deputy Arbiters
102 Time Control
112 Dates of rounds
122
001   1 m  GM Player Name              GER  12345678  2650  2650   1     7.5  w    2  b   15  w    3  ...
001   2 m  IM Other Player             AUT  87654321  2520  2520   2     7.0  b    1  w    8  b    5  ...
...
```

**Files to create:**
```
backend/
├── internal/export/
│   ├── trf.go              # TRF16/TRF06 generator
│   ├── dwz.go              # DWZ format
│   ├── pdf.go              # PDF generation
│   ├── html.go             # HTML export
│   └── chess_results.go    # Chess-Results integration
├── internal/submission/
│   ├── fide.go             # FIDE submission
│   └── dsb.go              # DSB submission
```

### Module 9: Notifications & Communication

**Features:**
- Email notifications (SendGrid/SES)
- Push notifications (web push)
- In-app notifications
- Tournament announcements
- Pairing notifications
- Game reminders
- Result confirmations
- Arbiter alerts

**Files to create:**
```
backend/
├── internal/notification/
│   ├── service.go
│   ├── email.go
│   ├── push.go
│   └── templates/
│       ├── pairing.html
│       ├── game_starting.html
│       └── ...
```

### Module 10: Fair Play (Basic)

**Features (MVP):**
- Connection monitoring
- Tab focus detection
- Basic time analysis
- Arbiter flagging system
- Manual review tools

**Note:** Advanced cheat detection (move analysis, pattern matching) is complex and should be deferred or partnered.

### Module 11: Team Tournaments (Mannschaftsmeisterschaft)

**Features:**
- Team management (roster, board order)
- Team registration for tournaments
- Lineup submission per round
- Rating order enforcement (configurable)
- Team pairing (Swiss or Round Robin)
- Match result calculation
- Board-by-board result entry
- Team standings with tiebreaks
- Bundesliga/regional league support

**Team Lineup Rules (German Mannschaftsordnung):**

```go
// Lineup validation rules
type LineupValidator struct {
    EnforceRatingOrder    bool    // Players ordered by rating (highest on board 1)
    RatingOrderTolerance  int     // Allowed ELO difference for swaps
    MinPlayers            int     // Minimum players required
    MaxPlayers            int     // Maximum players allowed
    AllowSubstitutes      bool    // Allow reserve players
    SubstituteRules       string  // "any_board", "lower_boards_only"
}

func (v *LineupValidator) Validate(lineup []LineupEntry, teamRoster []TeamMember) error {
    // 1. Check all players are on team roster
    // 2. Check rating order (with tolerance)
    // 3. Check no duplicate players
    // 4. Check board count matches tournament settings
    // 5. Check substitute rules
    return nil
}
```

**Team Pairing System:**

```go
// Team Swiss pairing (similar to individual but with teams)
type TeamPairer struct {
    engine   PairingEngine  // JaVaFo or bbpPairings
    settings TeamTournamentSettings
}

func (p *TeamPairer) GeneratePairings(tournament *Tournament, round int) ([]TeamMatch, error) {
    // 1. Build team standings (match points, board points)
    // 2. Generate TRF with teams as "players"
    // 3. Run pairing engine
    // 4. Parse results into team matches
    // 5. Assign home/away (color equivalent)
    return matches, nil
}
```

**Team Match Workflow:**

```
1. Round created
       │
       ▼
2. Team pairings generated
       │
       ▼
3. Lineup submission opens
       │
   ┌───┴───┐
   ▼       ▼
4a. Team A   4b. Team B
   submits      submits
   lineup       lineup
   │            │
   └─────┬──────┘
         ▼
5. Lineups locked (deadline or manual)
         │
         ▼
6. Board pairings created
   (Player A1 vs Player B1, etc.)
         │
         ▼
7. Games played (OTB or online)
         │
         ▼
8. Results entered per board
         │
         ▼
9. Match result calculated
   (sum of board points → match points)
         │
         ▼
10. Team standings updated
```

**Files to create:**
```
backend/
├── internal/team/
│   ├── handler.go              # Team CRUD handlers
│   ├── service.go              # Team business logic
│   ├── repository.go           # Team data access
│   └── roster.go               # Roster management
├── internal/team_tournament/
│   ├── handler.go              # Team tournament handlers
│   ├── service.go              # Team tournament logic
│   ├── registration.go         # Team registration
│   ├── lineup/
│   │   ├── handler.go          # Lineup submission
│   │   ├── service.go          # Lineup logic
│   │   └── validator.go        # Lineup rules validation
│   ├── pairing/
│   │   ├── team_pairer.go      # Team pairing generation
│   │   └── match_creator.go    # Create board pairings
│   ├── match/
│   │   ├── handler.go          # Match management
│   │   ├── service.go          # Match logic
│   │   └── result.go           # Result calculation
│   └── standings/
│       ├── calculator.go       # Team standings
│       └── tiebreaks.go        # Team tiebreak systems
```

**Frontend Components:**
```
frontend/src/components/team/
├── TeamCard.tsx                # Team display card
├── TeamRoster.tsx              # Show team members
├── TeamRosterEditor.tsx        # Edit team roster
├── LineupSubmission.tsx        # Submit lineup for round
├── LineupDisplay.tsx           # View lineup
├── TeamMatchCard.tsx           # Team match display
├── TeamMatchBoards.tsx         # Board-by-board view
├── TeamStandings.tsx           # Team tournament standings
└── TeamResultEntry.tsx         # Enter board results
```

**Team Tiebreak Systems:**
```go
const (
    TeamTBBoardPoints     = "board_points"      // Total board points scored
    TeamTBBuchholz        = "buchholz"          // Sum of opponents' match points
    TeamTBSonnebornBerger = "sonneborn_berger"  // Weighted by opponent strength
    TeamTBDirectEncounter = "direct_encounter"  // Head-to-head result
    TeamTBWins            = "wins"              // Number of match wins
    TeamTBBerlin          = "berlin"            // Board points in won matches
    TeamTBMatchesPlayed   = "matches_played"    // Fewer forfeits = better
)
```

---

## Frontend Structure

```
frontend/
├── src/
│   ├── app/                          # Next.js App Router
│   │   ├── (auth)/
│   │   │   ├── login/page.tsx
│   │   │   ├── register/page.tsx
│   │   │   └── forgot-password/page.tsx
│   │   ├── (dashboard)/
│   │   │   ├── layout.tsx
│   │   │   ├── page.tsx              # Dashboard home
│   │   │   ├── tournaments/
│   │   │   │   ├── page.tsx          # List
│   │   │   │   ├── new/page.tsx      # Create
│   │   │   │   └── [id]/
│   │   │   │       ├── page.tsx      # Details
│   │   │   │       ├── pairings/page.tsx
│   │   │   │       ├── standings/page.tsx
│   │   │   │       ├── games/page.tsx
│   │   │   │       └── settings/page.tsx
│   │   │   ├── players/
│   │   │   │   ├── page.tsx          # Search
│   │   │   │   └── [id]/page.tsx     # Profile
│   │   │   └── organizations/
│   │   │       └── [id]/page.tsx
│   │   ├── (public)/
│   │   │   ├── page.tsx              # Landing
│   │   │   ├── tournaments/page.tsx   # Public list
│   │   │   └── t/[slug]/             # Public tournament view
│   │   │       ├── page.tsx
│   │   │       ├── register/page.tsx
│   │   │       └── live/page.tsx     # Live games
│   │   ├── arbiter/
│   │   │   └── [tournamentId]/
│   │   │       ├── page.tsx          # Arbiter console
│   │   │       ├── pairings/page.tsx
│   │   │       └── games/page.tsx
│   │   ├── play/
│   │   │   └── [gameId]/page.tsx     # Chess game UI
│   │   └── api/                      # Next.js API routes (proxy)
│   ├── components/
│   │   ├── ui/                       # shadcn/ui components
│   │   ├── chess/
│   │   │   ├── Board.tsx             # Interactive chessboard
│   │   │   ├── Clock.tsx             # Game clock
│   │   │   ├── MoveList.tsx
│   │   │   └── GameControls.tsx
│   │   ├── tournament/
│   │   │   ├── TournamentCard.tsx
│   │   │   ├── StandingsTable.tsx
│   │   │   ├── PairingsView.tsx
│   │   │   ├── CrossTable.tsx
│   │   │   └── RegistrationForm.tsx
│   │   ├── player/
│   │   │   ├── PlayerCard.tsx
│   │   │   ├── PlayerSearch.tsx
│   │   │   └── RatingBadge.tsx
│   │   └── arbiter/
│   │       ├── ArbiterConsole.tsx
│   │       ├── ResultEntry.tsx
│   │       └── PairingEditor.tsx
│   ├── lib/
│   │   ├── api.ts                    # API client
│   │   ├── websocket.ts              # WS client
│   │   ├── chess.ts                  # Chess utilities
│   │   └── utils.ts
│   ├── hooks/
│   │   ├── useAuth.ts
│   │   ├── useWebSocket.ts
│   │   ├── useGame.ts
│   │   └── useTournament.ts
│   ├── store/                        # Zustand stores
│   │   ├── auth.ts
│   │   ├── game.ts
│   │   └── tournament.ts
│   └── i18n/
│       ├── de.json                   # German
│       ├── en.json                   # English
│       └── config.ts
├── public/
│   ├── pieces/                       # Chess piece SVGs
│   └── sounds/                       # Move sounds
└── package.json
```

---

## Implementation Phases with Test Strategy

---

### Phase 1: Foundation (Weeks 1-4)

**Goal:** Basic project setup and core infrastructure

#### Week 1: Project Setup
- [ ] Initialize Go module with project structure
- [ ] Set up Next.js 14 with App Router
- [ ] Configure TypeScript, ESLint, Prettier
- [ ] Set up Tailwind CSS + shadcn/ui
- [ ] Create docker-compose.yml (PostgreSQL, Redis)
- [ ] Initialize database schema with migrations (golang-migrate)
- [ ] Configure environment variables (.env structure)

#### Week 2: Authentication Backend
- [ ] Implement user registration endpoint
- [ ] Implement login endpoint with JWT
- [ ] Implement refresh token rotation
- [ ] Implement password reset flow
- [ ] Create auth middleware
- [ ] Implement role-based access control (RBAC)
- [ ] Add rate limiting for auth endpoints

#### Week 3: Authentication Frontend + User Management
- [ ] Create login page with form validation
- [ ] Create registration page
- [ ] Implement auth context/store (Zustand)
- [ ] Create protected route wrapper
- [ ] Implement user profile page
- [ ] Create user settings page
- [ ] Add language preference (DE/EN)

#### Week 4: CI/CD & Infrastructure
- [ ] Set up GitHub Actions CI pipeline
- [ ] Configure linting and formatting checks
- [ ] Set up test runners (Go test, Jest/Vitest)
- [ ] Create Dockerfiles for backend and frontend
- [ ] Set up staging environment
- [ ] Configure database migrations in CI
- [ ] Create API documentation (OpenAPI/Swagger)

**Deliverable:** Users can register, login, manage profile

#### Phase 1 Test Strategy

**Unit Tests (Target: 80% coverage)**
```
backend/internal/auth/
├── service_test.go
│   ├── TestRegisterUser_Success
│   ├── TestRegisterUser_DuplicateEmail
│   ├── TestRegisterUser_InvalidEmail
│   ├── TestRegisterUser_WeakPassword
│   ├── TestLogin_Success
│   ├── TestLogin_WrongPassword
│   ├── TestLogin_UserNotFound
│   ├── TestRefreshToken_Valid
│   ├── TestRefreshToken_Expired
│   └── TestRefreshToken_Revoked
├── jwt_test.go
│   ├── TestGenerateToken
│   ├── TestValidateToken_Valid
│   ├── TestValidateToken_Expired
│   └── TestValidateToken_InvalidSignature
└── middleware_test.go
    ├── TestAuthMiddleware_ValidToken
    ├── TestAuthMiddleware_MissingToken
    ├── TestAuthMiddleware_InvalidToken
    └── TestRBACMiddleware_InsufficientPermissions
```

**Integration Tests**
```go
// backend/tests/integration/auth_test.go
func TestAuthFlow(t *testing.T) {
    // 1. Register new user
    // 2. Verify email (if enabled)
    // 3. Login and receive tokens
    // 4. Access protected endpoint
    // 5. Refresh token
    // 6. Logout (revoke tokens)
    // 7. Verify old token is invalid
}

func TestPasswordResetFlow(t *testing.T) {
    // 1. Request password reset
    // 2. Verify reset email sent (mock)
    // 3. Reset password with token
    // 4. Login with new password
    // 5. Verify old password doesn't work
}
```

**Frontend Tests (Vitest + React Testing Library)**
```typescript
// frontend/src/__tests__/auth/
├── LoginForm.test.tsx
│   ├── renders login form
│   ├── shows validation errors for empty fields
│   ├── shows error for invalid credentials
│   ├── redirects on successful login
│   └── handles network errors gracefully
├── RegisterForm.test.tsx
│   ├── renders registration form
│   ├── validates email format
│   ├── validates password strength
│   ├── shows error for existing email
│   └── redirects after successful registration
└── useAuth.test.tsx
    ├── provides auth context
    ├── persists auth state
    ├── handles token refresh
    └── clears state on logout
```

**E2E Tests (Playwright)**
```typescript
// e2e/auth.spec.ts
test.describe('Authentication', () => {
    test('complete registration flow', async ({ page }) => {
        await page.goto('/register');
        await page.fill('[name=email]', 'test@example.com');
        await page.fill('[name=password]', 'SecurePass123!');
        await page.fill('[name=confirmPassword]', 'SecurePass123!');
        await page.click('button[type=submit]');
        await expect(page).toHaveURL('/dashboard');
    });

    test('login with valid credentials', async ({ page }) => {
        // ... test implementation
    });

    test('shows error for invalid login', async ({ page }) => {
        // ... test implementation
    });
});
```

**Security Tests**
- [ ] SQL injection testing on all inputs
- [ ] XSS testing on user-generated content
- [ ] CSRF token validation
- [ ] Rate limiting verification (brute force protection)
- [ ] Password hashing verification (bcrypt cost factor)
- [ ] JWT token security (algorithm, expiry)

**Phase 1 Exit Criteria:**
- [ ] All unit tests pass (≥80% coverage)
- [ ] All integration tests pass
- [ ] All E2E tests pass
- [ ] Security scan passes (no critical/high issues)
- [ ] API documentation complete
- [ ] Code review completed
- [ ] Staging deployment successful

---

### Phase 2: Player Database (Weeks 5-7)

**Goal:** Player management with federation sync

#### Week 5: Player Core
- [ ] Implement players table and repository
- [ ] Create player CRUD endpoints
- [ ] Implement player search (full-text PostgreSQL)
- [ ] Create player profile page
- [ ] Implement rating display component
- [ ] Add player card component
- [ ] Create player list with pagination

#### Week 6: FIDE Integration
- [ ] Download FIDE XML rating list
- [ ] Implement XML parser (streaming for large files)
- [ ] Create FIDE sync worker job
- [ ] Implement player matching (by FIDE ID)
- [ ] Handle new players from FIDE
- [ ] Track rating history
- [ ] Create sync status dashboard

#### Week 7: DSB Integration
- [ ] Register for DSB API access
- [ ] Implement DSB API client
- [ ] Create DSB sync worker job
- [ ] Match players by DSB PKZ
- [ ] Import DWZ ratings
- [ ] Handle club associations
- [ ] Create unified player search

**Deliverable:** Searchable player database with FIDE/DWZ data

#### Phase 2 Test Strategy

**Unit Tests**
```
backend/internal/player/
├── service_test.go
│   ├── TestCreatePlayer_Success
│   ├── TestCreatePlayer_DuplicateFideId
│   ├── TestUpdatePlayer_Success
│   ├── TestSearchPlayers_ByName
│   ├── TestSearchPlayers_ByFideId
│   ├── TestSearchPlayers_ByDwz
│   └── TestSearchPlayers_Pagination
├── repository_test.go
│   ├── TestFindByFideId
│   ├── TestFindByDsbId
│   └── TestFullTextSearch
└── search_test.go
    ├── TestSearchQuery_Parsing
    ├── TestSearchQuery_Ranking
    └── TestSearchQuery_Filters

backend/internal/federation/fide/
├── importer_test.go
│   ├── TestParseXML_ValidFile
│   ├── TestParseXML_LargeFile (benchmark)
│   ├── TestParseXML_MalformedXML
│   ├── TestMatchPlayer_ExistingFideId
│   ├── TestMatchPlayer_NewPlayer
│   └── TestMatchPlayer_RatingUpdate
└── scheduler_test.go
    ├── TestScheduler_MonthlySync
    └── TestScheduler_RetryOnFailure

backend/internal/federation/dsb/
├── client_test.go
│   ├── TestGetPlayer_Success
│   ├── TestGetPlayer_NotFound
│   ├── TestGetPlayer_APIError
│   ├── TestGetClub_Success
│   └── TestParseCSVResponse
└── importer_test.go
    ├── TestSyncPlayer_NewPlayer
    ├── TestSyncPlayer_UpdateRating
    └── TestSyncPlayer_ClubChange
```

**Integration Tests**
```go
// backend/tests/integration/federation_test.go
func TestFIDEImport(t *testing.T) {
    // Use sample FIDE XML file (subset)
    // 1. Import players
    // 2. Verify player records created
    // 3. Verify ratings correct
    // 4. Re-import (update scenario)
    // 5. Verify rating history recorded
}

func TestDSBSync(t *testing.T) {
    // Use mock DSB API server
    // 1. Sync player by PKZ
    // 2. Verify DWZ imported
    // 3. Verify club association
    // 4. Test incremental update
}

func TestPlayerSearch(t *testing.T) {
    // Seed database with test players
    // 1. Search by exact name
    // 2. Search by partial name
    // 3. Search by FIDE ID
    // 4. Filter by rating range
    // 5. Filter by federation
    // 6. Pagination works correctly
}
```

**Data Validation Tests**
```go
func TestFIDEDataIntegrity(t *testing.T) {
    // After import:
    // - No duplicate FIDE IDs
    // - All ratings within valid range (0-3000)
    // - All federations are valid ISO codes
    // - Titles are valid FIDE titles
}

func TestPlayerMerge(t *testing.T) {
    // Test deduplication scenarios:
    // - Same FIDE ID, different names (married name change)
    // - Same name, different FIDE IDs (different people)
    // - Manual merge workflow
}
```

**Performance Tests**
```go
func BenchmarkFIDEImport(b *testing.B) {
    // Import full FIDE list (~500k players)
    // Target: < 10 minutes
}

func BenchmarkPlayerSearch(b *testing.B) {
    // Search with 500k players in DB
    // Target: < 100ms per search
}
```

**Phase 2 Exit Criteria:**
- [ ] FIDE import completes successfully
- [ ] DSB API integration works
- [ ] Player search returns results < 100ms
- [ ] Rating history tracked correctly
- [ ] All federation sync jobs run on schedule
- [ ] Integration tests pass
- [ ] Manual QA: search 10 known players

---

### Phase 3: Tournament Core (Weeks 8-12)

**Goal:** Full OTB tournament workflow

#### Week 8: Tournament CRUD
- [ ] Implement tournaments table and repository
- [ ] Create tournament CRUD endpoints
- [ ] Build tournament creation wizard (frontend)
- [ ] Implement tournament settings (time control, tiebreaks)
- [ ] Create tournament list page
- [ ] Build tournament detail page
- [ ] Implement tournament status workflow

#### Week 9: Registration System
- [ ] Implement participant registration
- [ ] Create registration form component
- [ ] Build participant list management
- [ ] Implement waitlist functionality
- [ ] Add bulk import (CSV)
- [ ] Create starting list with rankings
- [ ] Implement bye request handling

#### Week 10: Pairing Engine Integration
- [ ] Set up JaVaFo binary
- [ ] Implement TRF generator
- [ ] Create TRF parser
- [ ] Build JaVaFo wrapper service
- [ ] Implement pairing workflow
- [ ] Create pairing preview
- [ ] Build manual pairing override

#### Week 11: Results & Standings
- [ ] Implement result entry endpoint
- [ ] Create result entry form
- [ ] Build tiebreak calculator (top 15 tiebreaks)
- [ ] Implement standings calculation
- [ ] Create standings table component
- [ ] Build crosstable generator
- [ ] Add round management UI

#### Week 12: Tournament Workflow Polish
- [ ] Complete round workflow (pair → play → results)
- [ ] Implement tournament completion flow
- [ ] Add tournament cloning
- [ ] Create public tournament view
- [ ] Build tournament search/filter
- [ ] Add tournament calendar view

**Deliverable:** Run a complete OTB Swiss tournament

#### Phase 3 Test Strategy

**Unit Tests**
```
backend/internal/tournament/
├── service_test.go
│   ├── TestCreateTournament_Success
│   ├── TestCreateTournament_InvalidDates
│   ├── TestUpdateTournament_StatusTransitions
│   ├── TestDeleteTournament_WithParticipants
│   └── TestCloneTournament
├── registration_test.go
│   ├── TestRegisterPlayer_Success
│   ├── TestRegisterPlayer_TournamentFull
│   ├── TestRegisterPlayer_RegistrationClosed
│   ├── TestWithdrawPlayer_BeforeStart
│   ├── TestWithdrawPlayer_AfterStart
│   └── TestBulkImport_CSV
└── tiebreak/
    ├── buchholz_test.go
    │   ├── TestBuchholz_Standard
    │   ├── TestBuchholz_WithByes
    │   ├── TestBuchholz_Cut1
    │   └── TestBuchholz_Cut2
    ├── sonneborn_test.go
    │   ├── TestSonnebornBerger_Standard
    │   └── TestSonnebornBerger_WithDraws
    └── calculator_test.go
        ├── TestCalculateAllTiebreaks
        └── TestTiebreakOrder

backend/internal/pairing/
├── engine/
│   ├── javafo_test.go
│   │   ├── TestGeneratePairings_Round1
│   │   ├── TestGeneratePairings_MiddleRound
│   │   ├── TestGeneratePairings_OddPlayers
│   │   ├── TestGeneratePairings_WithByes
│   │   └── TestGeneratePairings_ForbiddenPairs
│   └── interface_test.go
├── trf/
│   ├── generator_test.go
│   │   ├── TestGenerateTRF_FullTournament
│   │   ├── TestGenerateTRF_Header
│   │   └── TestGenerateTRF_PlayerLines
│   ├── parser_test.go
│   │   ├── TestParseTRF_Valid
│   │   ├── TestParseTRF_Pairings
│   │   └── TestParseTRF_Errors
│   └── validator_test.go
└── validator_test.go
    ├── TestValidatePairing_ColorBalance
    ├── TestValidatePairing_NoRematch
    └── TestValidatePairing_PointGroups

backend/internal/result/
├── service_test.go
│   ├── TestEnterResult_ValidResult
│   ├── TestEnterResult_InvalidResult
│   ├── TestEnterResult_GameNotFound
│   ├── TestEnterResult_AlreadyEntered
│   └── TestCorrectResult_WithAudit
└── validator_test.go
    ├── TestValidateResult_Formats
    └── TestValidateResult_Forfeits

backend/internal/standings/
├── calculator_test.go
│   ├── TestCalculateStandings_AfterRound
│   ├── TestCalculateStandings_WithTiebreaks
│   └── TestCalculateStandings_Ranking
├── crosstable_test.go
│   ├── TestGenerateCrosstable_Swiss
│   └── TestGenerateCrosstable_RoundRobin
└── norms_test.go
    ├── TestCheckNorm_IM
    ├── TestCheckNorm_GM
    └── TestCalculatePerformance
```

**Integration Tests**
```go
// backend/tests/integration/tournament_test.go
func TestCompleteTournamentWorkflow(t *testing.T) {
    // 1. Create tournament
    // 2. Register 20 players
    // 3. Generate round 1 pairings
    // 4. Enter all round 1 results
    // 5. Verify standings after round 1
    // 6. Generate round 2 pairings
    // 7. Verify no rematches
    // 8. Complete all 7 rounds
    // 9. Verify final standings
    // 10. Verify tiebreaks calculated correctly
}

func TestPairingEngine_JaVaFo(t *testing.T) {
    // Test against known good pairings
    // Use FIDE test cases
}

func TestTRFRoundTrip(t *testing.T) {
    // Generate TRF → Parse TRF → Compare
}
```

**Pairing Verification Tests**
```go
// Test against official FIDE pairing examples
func TestFIDEPairingExamples(t *testing.T) {
    testCases := []struct {
        name     string
        trfInput string
        expected []Pairing
    }{
        {"FIDE Example 1", "testdata/fide_example_1.trf", ...},
        {"FIDE Example 2", "testdata/fide_example_2.trf", ...},
        // All official FIDE test cases
    }
}
```

**Frontend Tests**
```typescript
// frontend/src/__tests__/tournament/
├── TournamentWizard.test.tsx
├── ParticipantList.test.tsx
├── PairingsView.test.tsx
├── ResultEntry.test.tsx
├── StandingsTable.test.tsx
└── CrossTable.test.tsx
```

**E2E Tests**
```typescript
// e2e/tournament.spec.ts
test.describe('Tournament Workflow', () => {
    test('create tournament and register players', async ({ page }) => {
        // Create tournament via wizard
        // Register 5 players
        // Verify participant list
    });

    test('complete pairing workflow', async ({ page }) => {
        // Generate pairings
        // Verify pairings displayed
        // Enter results
        // Verify standings update
    });

    test('complete 3-round tournament', async ({ page }) => {
        // Full workflow test
    });
});
```

**Phase 3 Exit Criteria:**
- [ ] JaVaFo pairings match FIDE test cases
- [ ] All 15 core tiebreaks calculate correctly
- [ ] TRF export validates against FIDE format
- [ ] Complete tournament workflow tested E2E
- [ ] Performance: pairing < 5s for 200 players
- [ ] Manual QA: run simulated tournament

---

### Phase 4: Arbiter Tools (Weeks 13-14)

**Goal:** Professional arbiter experience

#### Week 13: Arbiter Console
- [ ] Create arbiter dashboard layout
- [ ] Implement real-time tournament status
- [ ] Build game status overview
- [ ] Create quick result entry interface
- [ ] Implement player status management
- [ ] Add round timer/schedule display
- [ ] Create announcement system

#### Week 14: Arbiter Actions
- [ ] Implement manual pairing override
- [ ] Create player withdrawal workflow
- [ ] Build bye assignment tool
- [ ] Implement result correction with audit
- [ ] Add multi-arbiter support
- [ ] Create mobile-optimized arbiter view
- [ ] Implement arbiter permissions

**Deliverable:** Full arbiter control panel

#### Phase 4 Test Strategy

**Unit Tests**
```
backend/internal/arbiter/
├── service_test.go
│   ├── TestManualPairingOverride
│   ├── TestWithdrawPlayer_MidRound
│   ├── TestWithdrawPlayer_BetweenRounds
│   ├── TestAssignBye_FullPoint
│   ├── TestAssignBye_HalfPoint
│   ├── TestCorrectResult_WithCascade
│   └── TestAuditLog_Actions
├── console_test.go
│   ├── TestGetTournamentStatus
│   ├── TestGetActiveGames
│   └── TestGetPendingResults
└── permissions_test.go
    ├── TestArbiterCanEditPairings
    ├── TestDeputyArbiterPermissions
    └── TestOrganizer Permissions
```

**Integration Tests**
```go
func TestArbiterWorkflow_PlayerWithdrawal(t *testing.T) {
    // 1. Tournament with 20 players, round 2 in progress
    // 2. Player withdraws mid-round
    // 3. Opponent gets forfeit win
    // 4. Player marked inactive
    // 5. Round 3 pairings exclude withdrawn player
    // 6. Standings recalculated
}

func TestResultCorrection(t *testing.T) {
    // 1. Enter result: 1-0
    // 2. Correct to 0-1
    // 3. Verify audit log
    // 4. Verify standings recalculated
    // 5. Verify tiebreaks updated
}

func TestMultiArbiterConcurrency(t *testing.T) {
    // Two arbiters editing simultaneously
    // Verify no race conditions
    // Verify audit trail for both
}
```

**E2E Tests**
```typescript
test.describe('Arbiter Console', () => {
    test('arbiter can override pairing', async ({ page }) => {
        // Login as arbiter
        // Navigate to pairings
        // Swap two players
        // Verify pairing updated
    });

    test('arbiter can enter results on mobile', async ({ page }) => {
        // Set mobile viewport
        // Navigate to arbiter console
        // Enter result via touch
        // Verify result saved
    });
});
```

**Phase 4 Exit Criteria:**
- [ ] All arbiter actions have audit logs
- [ ] Multi-arbiter concurrency safe
- [ ] Mobile view usable on tablet
- [ ] Manual QA: arbiter workflow test

---

### Phase 5: Chess Server MVP (Weeks 15-18)

**Goal:** Online game play

#### Week 15: WebSocket Infrastructure
- [ ] Set up WebSocket server (Gorilla)
- [ ] Implement connection management
- [ ] Create message protocol
- [ ] Build connection hub
- [ ] Implement authentication for WS
- [ ] Add connection heartbeat
- [ ] Create reconnection handling

#### Week 16: Chess Game Logic
- [ ] Integrate chess library (notnil/chess)
- [ ] Implement game state management
- [ ] Create move validation
- [ ] Build game state persistence (Redis)
- [ ] Implement game history
- [ ] Create PGN generator

#### Week 17: Clock & Real-time
- [ ] Implement chess clock logic
- [ ] Create clock synchronization
- [ ] Build timeout handling
- [ ] Implement time controls (standard, rapid, blitz)
- [ ] Add increment handling
- [ ] Create clock UI component

#### Week 18: Game Features
- [ ] Implement draw offers
- [ ] Create resignation handling
- [ ] Build abort (before move 2)
- [ ] Implement spectating
- [ ] Add move broadcast
- [ ] Create game reconnection UI
- [ ] Build game history viewer

**Deliverable:** Play chess games online

#### Phase 5 Test Strategy

**Unit Tests**
```
backend/internal/chess/
├── game_test.go
│   ├── TestNewGame_InitialPosition
│   ├── TestMakeMove_ValidMove
│   ├── TestMakeMove_InvalidMove
│   ├── TestMakeMove_NotYourTurn
│   ├── TestMakeMove_Checkmate
│   ├── TestMakeMove_Stalemate
│   ├── TestMakeMove_DrawByRepetition
│   ├── TestMakeMove_50MoveRule
│   └── TestMakeMove_InsufficientMaterial
├── clock_test.go
│   ├── TestClock_Decrement
│   ├── TestClock_Increment
│   ├── TestClock_Timeout
│   ├── TestClock_Pause
│   └── TestClock_Sync
├── move_test.go
│   ├── TestValidateMove_AllPieceTypes
│   ├── TestValidateMove_Castling
│   ├── TestValidateMove_EnPassant
│   ├── TestValidateMove_Promotion
│   └── TestValidateMove_Pin
└── pgn_test.go
    ├── TestGeneratePGN_CompleteGame
    ├── TestGeneratePGN_WithComments
    └── TestParsePGN

backend/internal/server/
├── websocket_test.go
│   ├── TestConnection_Establish
│   ├── TestConnection_Authenticate
│   ├── TestConnection_Heartbeat
│   └── TestConnection_Reconnect
├── hub_test.go
│   ├── TestHub_RegisterClient
│   ├── TestHub_UnregisterClient
│   └── TestHub_Broadcast
└── game_handler_test.go
    ├── TestHandler_JoinGame
    ├── TestHandler_MakeMove
    ├── TestHandler_Resign
    ├── TestHandler_DrawOffer
    └── TestHandler_Spectate
```

**Integration Tests**
```go
func TestCompleteOnlineGame(t *testing.T) {
    // 1. Create game
    // 2. Two players connect via WS
    // 3. Play moves alternating
    // 4. Verify clock decrements
    // 5. Reach checkmate
    // 6. Verify game ends correctly
    // 7. Verify PGN generated
}

func TestGameReconnection(t *testing.T) {
    // 1. Start game
    // 2. Player disconnects
    // 3. Clock continues
    // 4. Player reconnects
    // 5. Game state restored
    // 6. Continue playing
}

func TestClockTimeout(t *testing.T) {
    // 1. Start game with 1 minute each
    // 2. One player doesn't move
    // 3. Wait for timeout
    // 4. Verify game ends with timeout result
}

func TestSpectating(t *testing.T) {
    // 1. Start game
    // 2. Spectator joins
    // 3. Spectator receives moves
    // 4. Spectator count updated
    // 5. Spectator leaves
}
```

**Load Tests**
```go
func TestConcurrentGames(t *testing.T) {
    // Simulate 100 concurrent games
    // Each game: 2 players, moves every 2s
    // Target: all games complete without errors
}

func BenchmarkMoveProcessing(b *testing.B) {
    // Target: < 10ms per move
}
```

**Frontend Tests**
```typescript
// frontend/src/__tests__/chess/
├── Board.test.tsx
│   ├── renders initial position
│   ├── highlights legal moves
│   ├── handles piece drag
│   ├── shows move animation
│   └── displays check indicator
├── Clock.test.tsx
│   ├── displays time correctly
│   ├── counts down
│   ├── shows low time warning
│   └── handles timeout
├── useGame.test.tsx
│   ├── connects to WebSocket
│   ├── handles move messages
│   ├── handles reconnection
│   └── handles game end
└── GameControls.test.tsx
    ├── resign button works
    ├── draw offer button works
    └── abort button (move < 2)
```

**E2E Tests**
```typescript
test.describe('Online Chess', () => {
    test('two players can play a game', async ({ browser }) => {
        // Create two browser contexts (two players)
        // Both join same game
        // Play 10 moves alternating
        // Verify board synced
    });

    test('game handles disconnect gracefully', async ({ page, context }) => {
        // Start game
        // Kill network
        // Verify reconnection prompt
        // Restore network
        // Verify game continues
    });
});
```

**Phase 5 Exit Criteria:**
- [ ] All chess rules implemented correctly
- [ ] Clock accuracy within 100ms
- [ ] Reconnection works reliably
- [ ] 100 concurrent games supported
- [ ] Move latency < 100ms
- [ ] Manual QA: play 10 complete games

---

### Phase 6: Hybrid & Team Tournaments (Weeks 19-22)

**Goal:** Combine online + OTB, Team tournament support

#### Week 19: Hybrid Tournament Integration
- [ ] Link online games to tournament pairings
- [ ] Implement auto-result sync
- [ ] Create hybrid tournament mode
- [ ] Build live broadcast view
- [ ] Add game embedding in tournament view

#### Week 20: Team Tournament Core
- [ ] Implement team CRUD
- [ ] Create team roster management
- [ ] Build team registration for tournaments
- [ ] Implement lineup submission
- [ ] Add lineup validation rules

#### Week 21: Team Match Workflow
- [ ] Implement team pairing
- [ ] Create team match management
- [ ] Build board-by-board result entry
- [ ] Implement match result calculation
- [ ] Add team standings calculator

#### Week 22: Team Tournament Polish
- [ ] Create team standings display
- [ ] Build team crosstable
- [ ] Implement team tiebreaks
- [ ] Add Bundesliga-specific features
- [ ] Create team management UI

**Deliverable:** Run hybrid and team tournaments

#### Phase 6 Test Strategy

**Unit Tests**
```
backend/internal/team/
├── service_test.go
│   ├── TestCreateTeam
│   ├── TestAddPlayerToRoster
│   ├── TestRemovePlayerFromRoster
│   └── TestValidateRoster
└── roster_test.go
    ├── TestRosterBoardOrder
    └── TestRosterValidation

backend/internal/team_tournament/
├── lineup/
│   ├── validator_test.go
│   │   ├── TestValidateLineup_ValidOrder
│   │   ├── TestValidateLineup_RatingOrderViolation
│   │   ├── TestValidateLineup_PlayerNotOnRoster
│   │   ├── TestValidateLineup_DuplicatePlayer
│   │   └── TestValidateLineup_MissingBoards
│   └── service_test.go
├── pairing/
│   ├── team_pairer_test.go
│   │   ├── TestTeamPairing_Round1
│   │   ├── TestTeamPairing_ByMatchPoints
│   │   └── TestTeamPairing_OddTeams
│   └── match_creator_test.go
├── match/
│   └── result_test.go
│       ├── TestCalculateMatchResult_HomeWins
│       ├── TestCalculateMatchResult_Draw
│       └── TestCalculateMatchResult_DefaultBoard
└── standings/
    ├── calculator_test.go
    │   ├── TestTeamStandings_ByMatchPoints
    │   └── TestTeamStandings_WithTiebreaks
    └── tiebreaks_test.go
```

**Integration Tests**
```go
func TestCompleteTeamTournament(t *testing.T) {
    // 1. Create team tournament (8 boards)
    // 2. Register 8 teams
    // 3. Round 1: generate pairings
    // 4. Teams submit lineups
    // 5. Lineups validated and locked
    // 6. Board pairings created
    // 7. Enter all board results
    // 8. Match results calculated
    // 9. Team standings updated
    // 10. Complete all rounds
    // 11. Verify final standings
}

func TestHybridTournament(t *testing.T) {
    // Mix of OTB and online games
    // Online results auto-sync to tournament
}
```

**E2E Tests**
```typescript
test.describe('Team Tournament', () => {
    test('captain can submit lineup', async ({ page }) => {
        // Login as team captain
        // Navigate to lineup submission
        // Assign players to boards
        // Submit lineup
        // Verify lineup saved
    });

    test('complete team match workflow', async ({ page }) => {
        // Two teams with lineups
        // Enter board results
        // Verify match result calculated
    });
});
```

**Phase 6 Exit Criteria:**
- [ ] Lineup validation follows German rules
- [ ] Team pairing works correctly
- [ ] Match results calculate from board results
- [ ] Team standings with tiebreaks work
- [ ] Manual QA: run 8-team tournament

---

### Phase 7: Reporting & Submission (Weeks 23-24)

**Goal:** Federation integration for rating submission

#### Week 23: Export Formats
- [ ] Implement TRF16 export (FIDE)
- [ ] Create DWZ submission format
- [ ] Build ÖSB export
- [ ] Implement title norm report
- [ ] Add PDF generation (standings, crosstable)
- [ ] Create tournament certificate generator

#### Week 24: Direct Submission
- [ ] Implement DSB API submission
- [ ] Create Chess-Results integration
- [ ] Build submission status tracking
- [ ] Add error handling for submissions
- [ ] Create submission history

**Deliverable:** One-click federation reporting

#### Phase 7 Test Strategy

**Unit Tests**
```
backend/internal/export/
├── trf_test.go
│   ├── TestGenerateTRF16_Header
│   ├── TestGenerateTRF16_PlayerLines
│   ├── TestGenerateTRF16_RoundResults
│   ├── TestGenerateTRF16_Validation
│   └── TestGenerateTRF16_SpecialCases
├── dwz_test.go
│   ├── TestGenerateDWZFormat
│   └── TestValidateDWZRequirements
└── pdf_test.go
    ├── TestGenerateStandingsPDF
    └── TestGenerateCrosstablePDF

backend/internal/submission/
├── dsb_test.go
│   ├── TestSubmitToDSB_Success
│   ├── TestSubmitToDSB_ValidationError
│   └── TestSubmitToDSB_APIError
└── fide_test.go
```

**Integration Tests**
```go
func TestTRFValidation(t *testing.T) {
    // Generate TRF
    // Validate against FIDE TRF validator tool
    // Check all fields present
}

func TestDSBSubmission(t *testing.T) {
    // Use DSB sandbox/test API
    // Submit test tournament
    // Verify acceptance
}
```

**Phase 7 Exit Criteria:**
- [ ] TRF passes FIDE validation
- [ ] DWZ format accepted by DSB
- [ ] PDF reports look professional
- [ ] Submission workflow complete

---

### Phase 8: Polish & Beta (Weeks 25-28)

**Goal:** Production ready

#### Week 25: Remaining Features
- [ ] Implement remaining tiebreaks (90+)
- [ ] Add bbpPairings integration
- [ ] Create tournament templates
- [ ] Implement advanced search
- [ ] Add notification preferences

#### Week 26: Payment & Communication
- [ ] Integrate Stripe payments
- [ ] Add SEPA direct debit
- [ ] Implement email notifications
- [ ] Create notification center
- [ ] Add SMS notifications (optional)

#### Week 27: Performance & Security
- [ ] Performance optimization
- [ ] Database query optimization
- [ ] Add caching layer
- [ ] Security audit
- [ ] Penetration testing
- [ ] GDPR compliance review

#### Week 28: Beta Preparation
- [ ] Documentation (user guides)
- [ ] API documentation
- [ ] Onboarding flow
- [ ] Beta tester recruitment
- [ ] Support system setup
- [ ] Monitoring & alerting

**Deliverable:** Beta launch

#### Phase 8 Test Strategy

**Full Regression Suite**
- All previous phase tests
- Performance benchmarks
- Security scans
- Accessibility audit

**Load Testing**
```go
func TestLoadScenario_Tournament(t *testing.T) {
    // Simulate 10 concurrent tournaments
    // 200 players each
    // All generating pairings simultaneously
    // Target: no errors, < 30s pairing time
}

func TestLoadScenario_OnlineGames(t *testing.T) {
    // 500 concurrent games
    // Target: < 100ms move latency
}
```

**Security Testing**
- [ ] OWASP Top 10 scan
- [ ] SQL injection testing
- [ ] XSS testing
- [ ] CSRF testing
- [ ] Authentication bypass testing
- [ ] Rate limiting verification
- [ ] Session management testing

**Beta Testing Protocol**
1. Internal alpha (week 1): Team runs test tournaments
2. Closed beta (weeks 2-3): 5 selected clubs
3. Open beta (week 4): Public sign-up

**Phase 8 Exit Criteria:**
- [ ] All regression tests pass
- [ ] Load tests pass
- [ ] Security audit passed
- [ ] 5 real tournaments completed
- [ ] No critical bugs open
- [ ] Documentation complete

---

## File Structure Overview

```
openpairing/
├── backend/
│   ├── cmd/
│   │   ├── api/main.go               # Main API server
│   │   ├── worker/main.go            # Background worker
│   │   └── chess/main.go             # Chess server
│   ├── internal/
│   │   ├── auth/
│   │   ├── user/
│   │   ├── player/
│   │   ├── tournament/
│   │   ├── pairing/
│   │   ├── result/
│   │   ├── standings/
│   │   ├── chess/
│   │   ├── server/
│   │   ├── arbiter/
│   │   ├── federation/
│   │   ├── export/
│   │   ├── notification/
│   │   └── fairplay/
│   ├── pkg/
│   │   ├── database/
│   │   ├── redis/
│   │   ├── config/
│   │   └── logger/
│   ├── migrations/
│   ├── bin/                          # JaVaFo, bbpPairings binaries
│   ├── go.mod
│   └── Dockerfile
├── frontend/
│   ├── src/
│   ├── public/
│   ├── package.json
│   └── Dockerfile
├── docker-compose.yml
├── docker-compose.dev.yml
├── .github/
│   └── workflows/
│       ├── ci.yml
│       └── deploy.yml
├── docs/
│   ├── api.md
│   ├── architecture.md
│   └── federation-integration.md
├── LICENSE                           # AGPL-3.0
└── README.md
```

---

## Key Dependencies

### Backend (Go)
```go
// go.mod
require (
    github.com/gin-gonic/gin          // HTTP framework
    github.com/gorilla/websocket      // WebSocket
    github.com/jackc/pgx/v5           // PostgreSQL
    github.com/redis/go-redis/v9      // Redis
    github.com/golang-jwt/jwt/v5      // JWT
    github.com/notnil/chess           // Chess library
    github.com/go-playground/validator // Validation
    github.com/rs/zerolog             // Logging
    github.com/spf13/viper            // Config
    github.com/stripe/stripe-go/v76   // Payments
    golang.org/x/crypto               // Password hashing
)
```

### Frontend (Node.js)
```json
{
  "dependencies": {
    "next": "^14.0.0",
    "react": "^18.0.0",
    "typescript": "^5.0.0",
    "@tanstack/react-query": "^5.0.0",
    "zustand": "^4.0.0",
    "chess.js": "^1.0.0",
    "react-chessboard": "^4.0.0",
    "tailwindcss": "^3.0.0",
    "shadcn/ui": "latest",
    "next-intl": "^3.0.0",
    "socket.io-client": "^4.0.0",
    "@stripe/stripe-js": "^2.0.0"
  }
}
```

---

## Quality Assurance Summary

### Testing Pyramid

```
                    ┌─────────────┐
                    │    E2E      │  ~50 tests
                    │  (Playwright)│  Slow, expensive
                    ├─────────────┤
                    │ Integration │  ~150 tests
                    │   Tests     │  Medium speed
                    ├─────────────┤
                    │             │
                    │    Unit     │  ~500+ tests
                    │    Tests    │  Fast, cheap
                    │             │
                    └─────────────┘
```

### Test Coverage Targets

| Component | Target Coverage | Critical Paths |
|-----------|----------------|----------------|
| Auth | 90% | Login, token refresh |
| Players | 85% | Search, federation sync |
| Tournament | 90% | Pairing, results, standings |
| Chess Server | 95% | Move validation, clock |
| Team Tournament | 85% | Lineup, match results |
| Export | 80% | TRF, DWZ format |

### CI/CD Quality Gates

```yaml
# .github/workflows/ci.yml quality gates
quality_gates:
  - name: Unit Tests
    command: go test ./... -cover
    threshold: 80%

  - name: Integration Tests
    command: go test ./tests/integration/...
    required: true

  - name: Frontend Tests
    command: npm run test:coverage
    threshold: 75%

  - name: E2E Tests
    command: npx playwright test
    required: true

  - name: Security Scan
    command: gosec ./...
    max_issues: 0 (high/critical)

  - name: Lint
    command: golangci-lint run && npm run lint
    required: true

  - name: Build
    command: docker build .
    required: true
```

### Manual QA Checklist (Per Phase)

**Phase Completion Review:**
- [ ] All automated tests pass
- [ ] Code coverage meets threshold
- [ ] No security vulnerabilities
- [ ] Performance benchmarks met
- [ ] Documentation updated
- [ ] UI/UX review completed
- [ ] Accessibility check passed
- [ ] Cross-browser testing (Chrome, Firefox, Safari)
- [ ] Mobile responsiveness verified
- [ ] German localization reviewed

### Bug Severity Classification

| Severity | Definition | Response Time |
|----------|------------|---------------|
| Critical | System unusable, data loss | 4 hours |
| High | Major feature broken | 24 hours |
| Medium | Feature impaired but usable | 1 week |
| Low | Minor issue, workaround exists | Next sprint |

---

## Success Criteria

### MVP (Month 6)
- [ ] Run a 50-player Swiss tournament OTB
- [ ] Run an 8-team Mannschaftsturnier
- [ ] Import players from FIDE and DSB
- [ ] Generate FIDE TRF export
- [ ] Submit DWZ to DSB
- [ ] 15+ tiebreak options working
- [ ] Mobile-responsive UI
- [ ] 80% test coverage

### Beta (Month 7)
- [ ] Run hybrid tournaments (online + OTB)
- [ ] 100 concurrent online games
- [ ] Complete team tournament workflow
- [ ] 90+ tiebreak options
- [ ] German localization complete
- [ ] 5 beta customers running real tournaments
- [ ] 85% test coverage

### Launch (Month 8)
- [ ] FIDE endorsement application submitted
- [ ] Payment processing live
- [ ] 99.9% uptime SLA
- [ ] Public API documented
- [ ] 20+ active organizations
- [ ] Security audit passed
- [ ] GDPR compliant

---

## Risk Register

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| JaVaFo integration issues | High | Medium | Have bbpPairings as backup |
| DSB API changes | Medium | Low | Abstract federation layer |
| Performance with 500+ players | Medium | Medium | Early load testing |
| FIDE endorsement delays | High | Medium | Focus on OTB first |
| Chess server scaling | High | Medium | Redis cluster, horizontal scaling |
| Cheat detection complexity | Medium | High | Defer to phase 3, partner option |

---

## Definition of Done

A feature is considered "Done" when:

1. **Code Complete**
   - [ ] Implementation matches acceptance criteria
   - [ ] Code follows style guide
   - [ ] No TODO comments left
   - [ ] Error handling implemented

2. **Tests Written**
   - [ ] Unit tests with ≥80% coverage
   - [ ] Integration tests for happy path
   - [ ] Edge cases covered
   - [ ] E2E test if user-facing

3. **Documentation**
   - [ ] API documentation updated
   - [ ] Code comments for complex logic
   - [ ] User guide updated (if applicable)

4. **Review Complete**
   - [ ] Code review approved
   - [ ] Security review (if auth/data related)
   - [ ] UI/UX review (if frontend)

5. **Deployed**
   - [ ] CI pipeline passes
   - [ ] Deployed to staging
   - [ ] Smoke tests pass
   - [ ] No regression in existing features
