-- WAM Enhancements: Joker System, Variable Group Sizes, U12 Handling

-- Phase 1: Joker System
-- Add joker_points to tournament_participants for WAM tournaments
ALTER TABLE tournament_participants
    ADD COLUMN joker_points INTEGER DEFAULT 0
    CHECK (joker_points IN (0, 50, 100, 150));

-- Add joker tracking and effective rating to wam_group_members
ALTER TABLE wam_group_members
    ADD COLUMN joker_points INTEGER DEFAULT 0,
    ADD COLUMN effective_rating INTEGER;

-- Phase 2: Variable Group Sizes (configurable, default: 4 and 6)
ALTER TABLE wam_tournament_config
    ADD COLUMN default_group_size INTEGER DEFAULT 4,
    ADD COLUMN allowed_group_sizes INTEGER[] DEFAULT '{4,6}';

-- Add target_size per group (allows different sizes in same tournament)
ALTER TABLE wam_groups
    ADD COLUMN target_size INTEGER DEFAULT 4;

-- Update existing groups to have target_size from config
UPDATE wam_groups g
SET target_size = COALESCE(
    (SELECT group_size FROM wam_tournament_config c WHERE c.tournament_id = g.tournament_id),
    4
);

-- Index for efficient joker queries
CREATE INDEX idx_tournament_participants_joker ON tournament_participants(tournament_id, joker_points)
    WHERE joker_points > 0;
