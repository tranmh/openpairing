-- Registration Settings: Self-service pre-registration and check-in

-- Add registration_settings JSONB to tournaments (follows bye_settings pattern)
ALTER TABLE tournaments
ADD COLUMN IF NOT EXISTS registration_settings JSONB DEFAULT '{}';

CREATE INDEX IF NOT EXISTS idx_tournaments_registration_settings
ON tournaments USING gin(registration_settings) WHERE registration_settings != '{}';

-- Guest pre-registrations table (for users without accounts)
CREATE TABLE tournament_guest_registrations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tournament_id UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,

    -- Guest identification
    email VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,

    -- Optional federation IDs for lookup
    fide_id VARCHAR(20),
    dsb_id VARCHAR(20),
    oesb_id VARCHAR(20),
    ssb_id VARCHAR(20),

    -- Resolved player (after federation lookup or manual approval)
    player_id UUID REFERENCES players(id) ON DELETE SET NULL,

    -- Registration flow
    -- Values: pre_registered, pending_approval, approved, checked_in, converted, rejected, expired, no_show
    registration_status VARCHAR(50) DEFAULT 'pre_registered' NOT NULL,

    -- Check-in tracking
    check_in_code VARCHAR(20),
    checked_in_at TIMESTAMPTZ,
    checked_in_by UUID REFERENCES users(id),
    check_in_method VARCHAR(50), -- self_service, qr_code, organizer

    -- Approval tracking
    approved_at TIMESTAMPTZ,
    approved_by UUID REFERENCES users(id),
    rejection_reason TEXT,

    -- Metadata
    notes TEXT,

    -- Timestamps
    pre_registered_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,

    UNIQUE(tournament_id, email)
);

CREATE INDEX idx_guest_registrations_tournament ON tournament_guest_registrations(tournament_id);
CREATE INDEX idx_guest_registrations_email ON tournament_guest_registrations(email);
CREATE INDEX idx_guest_registrations_status ON tournament_guest_registrations(tournament_id, registration_status);
CREATE INDEX idx_guest_registrations_check_in_code ON tournament_guest_registrations(check_in_code) WHERE check_in_code IS NOT NULL;

-- Add check-in fields to existing tournament_participants
ALTER TABLE tournament_participants
ADD COLUMN IF NOT EXISTS pre_registered_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS approved_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS approved_by UUID REFERENCES users(id),
ADD COLUMN IF NOT EXISTS checked_in_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS checked_in_by UUID REFERENCES users(id),
ADD COLUMN IF NOT EXISTS check_in_method VARCHAR(50),
ADD COLUMN IF NOT EXISTS check_in_code VARCHAR(20),
ADD COLUMN IF NOT EXISTS registration_source VARCHAR(50) DEFAULT 'organizer';
-- registration_source values: organizer, self_service, guest_converted

CREATE INDEX IF NOT EXISTS idx_participants_check_in_code ON tournament_participants(check_in_code) WHERE check_in_code IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_participants_checked_in ON tournament_participants(tournament_id, checked_in_at) WHERE checked_in_at IS NOT NULL;

-- Invite tokens for private tournaments
CREATE TABLE tournament_invite_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tournament_id UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    name VARCHAR(100), -- Optional label for the token
    max_uses INTEGER, -- NULL = unlimited
    uses_count INTEGER DEFAULT 0 NOT NULL,
    valid_until TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,

    UNIQUE(token_hash)
);

CREATE INDEX idx_invite_tokens_tournament ON tournament_invite_tokens(tournament_id);
CREATE INDEX idx_invite_tokens_hash ON tournament_invite_tokens(token_hash);

-- Update trigger for guest registrations
CREATE OR REPLACE FUNCTION update_guest_registration_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_guest_registration_updated_at
    BEFORE UPDATE ON tournament_guest_registrations
    FOR EACH ROW
    EXECUTE FUNCTION update_guest_registration_updated_at();
