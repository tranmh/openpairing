-- Email Verification Tokens
CREATE TABLE email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(token_hash)
);

CREATE INDEX idx_email_verification_tokens_user ON email_verification_tokens(user_id);
CREATE INDEX idx_email_verification_tokens_hash ON email_verification_tokens(token_hash);

-- Tournament Waitlist
CREATE TABLE tournament_waitlist (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tournament_id UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    starting_rating INTEGER,
    notes TEXT,
    notified_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tournament_id, player_id)
);

CREATE INDEX idx_waitlist_tournament ON tournament_waitlist(tournament_id);
CREATE INDEX idx_waitlist_position ON tournament_waitlist(tournament_id, position);
CREATE INDEX idx_waitlist_player ON tournament_waitlist(player_id);

-- Add current_round and participant_count columns to tournaments if not exists
-- These are denormalized counters for performance
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tournaments' AND column_name = 'current_round') THEN
        ALTER TABLE tournaments ADD COLUMN current_round INTEGER DEFAULT 0;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tournaments' AND column_name = 'participant_count') THEN
        ALTER TABLE tournaments ADD COLUMN participant_count INTEGER DEFAULT 0;
    END IF;
END
$$;
