-- WAM (Württembergische Amateurmeisterschaft) Tournament Tables

-- WAM Tournament Configuration
CREATE TABLE wam_tournament_config (
    tournament_id UUID PRIMARY KEY REFERENCES tournaments(id) ON DELETE CASCADE,
    group_size INTEGER NOT NULL DEFAULT 4,
    assignment_method VARCHAR(20) NOT NULL DEFAULT 'sequential', -- sequential, serpentine, random, manual
    enable_playoffs BOOLEAN DEFAULT FALSE,
    playoffs_top_n INTEGER DEFAULT 1,
    third_place_match BOOLEAN DEFAULT FALSE,
    youth_age_limit INTEGER DEFAULT 12,
    current_phase VARCHAR(20) DEFAULT 'setup', -- setup, group, playoff, complete
    tiebreaks JSONB DEFAULT '["direct_encounter", "sonneborn_berger", "buchholz"]',
    season VARCHAR(20), -- e.g., "2025/2026"
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- WAM Groups
CREATE TABLE wam_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tournament_id UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    name VARCHAR(10) NOT NULL, -- "A", "B", "C", etc.
    group_number INTEGER NOT NULL,
    rounds_total INTEGER NOT NULL DEFAULT 0,
    rounds_played INTEGER DEFAULT 0,
    is_complete BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tournament_id, group_number),
    UNIQUE(tournament_id, name)
);

CREATE INDEX idx_wam_groups_tournament ON wam_groups(tournament_id);

-- WAM Group Members (players assigned to groups)
CREATE TABLE wam_group_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES wam_groups(id) ON DELETE CASCADE,
    participant_id UUID NOT NULL REFERENCES tournament_participants(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL,
    points DECIMAL(4,1) DEFAULT 0,
    wins INTEGER DEFAULT 0,
    draws INTEGER DEFAULT 0,
    losses INTEGER DEFAULT 0,
    is_group_winner BOOLEAN DEFAULT FALSE,
    qualified_final BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(group_id, participant_id),
    UNIQUE(group_id, sort_order)
);

CREATE INDEX idx_wam_group_members_group ON wam_group_members(group_id);
CREATE INDEX idx_wam_group_members_participant ON wam_group_members(participant_id);

-- WAM Group Games (links games to groups for easy querying)
CREATE TABLE wam_group_games (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES wam_groups(id) ON DELETE CASCADE,
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    round_in_group INTEGER NOT NULL, -- Round number within the group (1, 2, 3, etc.)
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(group_id, game_id)
);

CREATE INDEX idx_wam_group_games_group ON wam_group_games(group_id);
CREATE INDEX idx_wam_group_games_game ON wam_group_games(game_id);

-- WAM Playoff Bracket
CREATE TABLE wam_playoff_matches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tournament_id UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    round_name VARCHAR(20) NOT NULL, -- "quarterfinal", "semifinal", "final", "third_place"
    match_number INTEGER NOT NULL,
    player1_participant_id UUID REFERENCES tournament_participants(id) ON DELETE SET NULL,
    player2_participant_id UUID REFERENCES tournament_participants(id) ON DELETE SET NULL,
    player1_source VARCHAR(50), -- e.g., "group_A_1" or "semifinal_1_winner"
    player2_source VARCHAR(50),
    winner_participant_id UUID REFERENCES tournament_participants(id) ON DELETE SET NULL,
    loser_participant_id UUID REFERENCES tournament_participants(id) ON DELETE SET NULL,
    game_id UUID REFERENCES games(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tournament_id, round_name, match_number)
);

CREATE INDEX idx_wam_playoff_tournament ON wam_playoff_matches(tournament_id);

-- WAM Qualifications (for annual finals tracking)
CREATE TABLE wam_qualifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tournament_id UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    participant_id UUID NOT NULL REFERENCES tournament_participants(id) ON DELETE CASCADE,
    player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    group_name VARCHAR(10) NOT NULL,
    group_rank INTEGER NOT NULL,
    season VARCHAR(20) NOT NULL, -- e.g., "2025/2026"
    qualified_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tournament_id, participant_id)
);

CREATE INDEX idx_wam_qualifications_player ON wam_qualifications(player_id);
CREATE INDEX idx_wam_qualifications_season ON wam_qualifications(season);

-- Triggers for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_wam_config_updated_at BEFORE UPDATE ON wam_tournament_config
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_wam_groups_updated_at BEFORE UPDATE ON wam_groups
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_wam_group_members_updated_at BEFORE UPDATE ON wam_group_members
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_wam_playoff_updated_at BEFORE UPDATE ON wam_playoff_matches
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
