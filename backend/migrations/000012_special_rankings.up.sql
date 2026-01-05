-- Special Rankings (Sonderwertungen) for prize eligibility
CREATE TABLE special_rankings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tournament_id UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    display_name VARCHAR(100) NOT NULL,

    -- Filter criteria (null = no filter on this criterion)
    min_birth_year INTEGER,
    max_birth_year INTEGER,
    min_rating INTEGER,
    max_rating INTEGER,
    gender VARCHAR(10),

    sort_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_special_rankings_tournament ON special_rankings(tournament_id);
CREATE INDEX idx_special_rankings_active ON special_rankings(tournament_id, is_active);
