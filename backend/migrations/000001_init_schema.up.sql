-- OpenPairing Initial Schema Migration
-- Version: 000001
-- Description: Creates all core tables for the chess tournament management platform

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm"; -- For fuzzy text search

-- ==========================================
-- USERS & AUTHENTICATION
-- ==========================================

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    role VARCHAR(50) NOT NULL DEFAULT 'player', -- player, arbiter, organizer, admin
    preferred_language VARCHAR(10) DEFAULT 'de',
    email_verified BOOLEAN DEFAULT FALSE,
    email_verified_at TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);

-- Refresh tokens for JWT rotation
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(token_hash)
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_hash ON refresh_tokens(token_hash);

-- Password reset tokens
CREATE TABLE password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(token_hash)
);

-- ==========================================
-- CLUBS
-- ==========================================

CREATE TABLE clubs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    short_name VARCHAR(50),
    federation VARCHAR(10), -- GER, AUT, SUI, etc.
    dsb_club_id VARCHAR(20), -- DSB Vereinsnummer (ZPS)
    oesb_club_id VARCHAR(20),
    city VARCHAR(100),
    country VARCHAR(3) DEFAULT 'DEU',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_clubs_name ON clubs USING gin(name gin_trgm_ops);
CREATE INDEX idx_clubs_dsb_id ON clubs(dsb_club_id);

-- ==========================================
-- PLAYERS
-- ==========================================

CREATE TABLE players (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,

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
    club_id UUID REFERENCES clubs(id) ON DELETE SET NULL,
    federation VARCHAR(10),           -- GER, AUT, SUI, etc.

    -- Metadata
    ratings_updated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_players_fide_id ON players(fide_id);
CREATE INDEX idx_players_dsb_id ON players(dsb_id);
CREATE INDEX idx_players_oesb_id ON players(oesb_id);
CREATE INDEX idx_players_name ON players(last_name, first_name);
CREATE INDEX idx_players_name_search ON players USING gin((first_name || ' ' || last_name) gin_trgm_ops);
CREATE INDEX idx_players_club ON players(club_id);
CREATE INDEX idx_players_user ON players(user_id);

-- ==========================================
-- ORGANIZATIONS
-- ==========================================

CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50), -- federation, club, private
    slug VARCHAR(100) UNIQUE,
    logo_url VARCHAR(500),
    website VARCHAR(500),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE organization_members (
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL, -- owner, admin, arbiter
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (organization_id, user_id)
);

-- ==========================================
-- TOURNAMENTS
-- ==========================================

CREATE TABLE tournaments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,

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
CREATE INDEX idx_tournaments_org ON tournaments(organization_id);
CREATE INDEX idx_tournaments_slug ON tournaments(slug);
CREATE INDEX idx_tournaments_type ON tournaments(type);

-- Tournament Sections (for multi-section events)
CREATE TABLE tournament_sections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tournament_id UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    rating_min INTEGER,
    rating_max INTEGER,
    age_min INTEGER,
    age_max INTEGER,
    gender VARCHAR(10),
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_sections_tournament ON tournament_sections(tournament_id);

-- ==========================================
-- TOURNAMENT PARTICIPANTS
-- ==========================================

CREATE TABLE tournament_participants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tournament_id UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    section_id UUID REFERENCES tournament_sections(id) ON DELETE SET NULL,
    player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,

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

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(tournament_id, player_id)
);

CREATE INDEX idx_participants_tournament ON tournament_participants(tournament_id);
CREATE INDEX idx_participants_player ON tournament_participants(player_id);
CREATE INDEX idx_participants_points ON tournament_participants(tournament_id, points DESC);
CREATE INDEX idx_participants_active ON tournament_participants(tournament_id, is_active);

-- ==========================================
-- TOURNAMENT ROUNDS
-- ==========================================

CREATE TABLE tournament_rounds (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tournament_id UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
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

    created_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(tournament_id, round_number)
);

CREATE INDEX idx_rounds_tournament ON tournament_rounds(tournament_id);
CREATE INDEX idx_rounds_status ON tournament_rounds(tournament_id, status);

-- ==========================================
-- GAMES/PAIRINGS
-- ==========================================

CREATE TABLE games (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tournament_id UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    round_id UUID NOT NULL REFERENCES tournament_rounds(id) ON DELETE CASCADE,
    section_id UUID REFERENCES tournament_sections(id) ON DELETE SET NULL,

    -- Pairing
    board_number INTEGER NOT NULL,
    white_participant_id UUID REFERENCES tournament_participants(id) ON DELETE SET NULL,
    black_participant_id UUID REFERENCES tournament_participants(id) ON DELETE SET NULL,

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
CREATE INDEX idx_games_round ON games(round_id);
CREATE INDEX idx_games_white ON games(white_participant_id);
CREATE INDEX idx_games_black ON games(black_participant_id);
CREATE INDEX idx_games_status ON games(status);

-- Forbidden Pairs (players who shouldn't be paired)
CREATE TABLE forbidden_pairs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tournament_id UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    player1_id UUID NOT NULL REFERENCES tournament_participants(id) ON DELETE CASCADE,
    player2_id UUID NOT NULL REFERENCES tournament_participants(id) ON DELETE CASCADE,
    reason VARCHAR(255),              -- same_club, siblings, requested
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_forbidden_tournament ON forbidden_pairs(tournament_id);

-- ==========================================
-- ONLINE CHESS GAMES
-- ==========================================

CREATE TABLE chess_games (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Players
    white_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    black_user_id UUID REFERENCES users(id) ON DELETE SET NULL,

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
CREATE INDEX idx_chess_games_white ON chess_games(white_user_id);
CREATE INDEX idx_chess_games_black ON chess_games(black_user_id);

-- ==========================================
-- TEAMS
-- ==========================================

CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    club_id UUID REFERENCES clubs(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    short_name VARCHAR(50),

    -- For league play (Bundesliga, etc.)
    division VARCHAR(100),            -- 1. Bundesliga, 2. Bundesliga, etc.
    region VARCHAR(100),              -- For regional leagues

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_teams_club ON teams(club_id);
CREATE INDEX idx_teams_name ON teams USING gin(name gin_trgm_ops);

-- Team Members (players assigned to a team)
CREATE TABLE team_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,

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
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tournament_id UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,

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

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(tournament_id, team_id)
);

CREATE INDEX idx_team_participants_tournament ON team_tournament_participants(tournament_id);
CREATE INDEX idx_team_participants_team ON team_tournament_participants(team_id);

-- Team Lineups (per round)
CREATE TABLE team_lineups (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tournament_id UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    round_id UUID NOT NULL REFERENCES tournament_rounds(id) ON DELETE CASCADE,
    team_participant_id UUID NOT NULL REFERENCES team_tournament_participants(id) ON DELETE CASCADE,

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

CREATE INDEX idx_team_lineups_round ON team_lineups(round_id);
CREATE INDEX idx_team_lineups_participant ON team_lineups(team_participant_id);

-- Team Matches (pairing between two teams)
CREATE TABLE team_matches (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tournament_id UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    round_id UUID NOT NULL REFERENCES tournament_rounds(id) ON DELETE CASCADE,

    match_number INTEGER NOT NULL,

    -- Teams
    home_team_id UUID REFERENCES team_tournament_participants(id) ON DELETE SET NULL,
    away_team_id UUID REFERENCES team_tournament_participants(id) ON DELETE SET NULL,

    -- Result
    home_match_points DECIMAL(3,1),   -- 2, 1, or 0
    away_match_points DECIMAL(3,1),
    home_board_points DECIMAL(4,1),   -- Sum of board results (e.g., 4.5)
    away_board_points DECIMAL(4,1),

    status VARCHAR(50) DEFAULT 'scheduled',

    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_team_matches_round ON team_matches(tournament_id, round_id);
CREATE INDEX idx_team_matches_home ON team_matches(home_team_id);
CREATE INDEX idx_team_matches_away ON team_matches(away_team_id);

-- Team Match Boards (individual games within a team match)
CREATE TABLE team_match_boards (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_match_id UUID NOT NULL REFERENCES team_matches(id) ON DELETE CASCADE,
    game_id UUID REFERENCES games(id) ON DELETE SET NULL, -- Links to the individual game

    board_number INTEGER NOT NULL,

    -- Players (from lineup)
    home_player_id UUID REFERENCES players(id) ON DELETE SET NULL,
    away_player_id UUID REFERENCES players(id) ON DELETE SET NULL,
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

-- ==========================================
-- RATING HISTORY
-- ==========================================

CREATE TABLE rating_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    rating_type VARCHAR(20) NOT NULL, -- fide_standard, fide_rapid, fide_blitz, dwz, oesb
    old_rating INTEGER,
    new_rating INTEGER NOT NULL,
    tournament_id UUID REFERENCES tournaments(id) ON DELETE SET NULL,
    change_date DATE NOT NULL,
    source VARCHAR(50) NOT NULL,      -- sync, tournament_result
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_rating_history_player ON rating_history(player_id);
CREATE INDEX idx_rating_history_type ON rating_history(player_id, rating_type);
CREATE INDEX idx_rating_history_date ON rating_history(change_date);

-- ==========================================
-- AUDIT LOG
-- ==========================================

CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID,
    old_value JSONB,
    new_value JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_audit_log_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_log_user ON audit_log(user_id);
CREATE INDEX idx_audit_log_action ON audit_log(action);
CREATE INDEX idx_audit_log_time ON audit_log(created_at);

-- ==========================================
-- NOTIFICATIONS
-- ==========================================

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,        -- pairing_published, game_starting, result_entered
    title VARCHAR(255) NOT NULL,
    message TEXT,
    data JSONB,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_notifications_user ON notifications(user_id);
CREATE INDEX idx_notifications_unread ON notifications(user_id, read_at) WHERE read_at IS NULL;

-- ==========================================
-- FEDERATION SYNC LOG
-- ==========================================

CREATE TABLE federation_sync_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    federation VARCHAR(20) NOT NULL,  -- fide, dsb, oesb
    sync_type VARCHAR(50) NOT NULL,   -- full, incremental
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    players_created INTEGER DEFAULT 0,
    players_updated INTEGER DEFAULT 0,
    players_total INTEGER DEFAULT 0,
    status VARCHAR(50) NOT NULL,      -- running, completed, failed
    error_message TEXT,
    metadata JSONB
);

CREATE INDEX idx_federation_sync_federation ON federation_sync_log(federation);
CREATE INDEX idx_federation_sync_status ON federation_sync_log(status);

-- ==========================================
-- TRIGGERS FOR updated_at
-- ==========================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_players_updated_at BEFORE UPDATE ON players
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_clubs_updated_at BEFORE UPDATE ON clubs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_organizations_updated_at BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tournaments_updated_at BEFORE UPDATE ON tournaments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_participants_updated_at BEFORE UPDATE ON tournament_participants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_teams_updated_at BEFORE UPDATE ON teams
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_team_participants_updated_at BEFORE UPDATE ON team_tournament_participants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
