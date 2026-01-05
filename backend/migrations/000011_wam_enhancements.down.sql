-- Rollback WAM Enhancements

-- Drop index
DROP INDEX IF EXISTS idx_tournament_participants_joker;

-- Remove variable group sizes from wam_groups
ALTER TABLE wam_groups DROP COLUMN IF EXISTS target_size;

-- Remove variable group sizes from wam_tournament_config
ALTER TABLE wam_tournament_config DROP COLUMN IF EXISTS allowed_group_sizes;
ALTER TABLE wam_tournament_config DROP COLUMN IF EXISTS default_group_size;

-- Remove joker tracking from wam_group_members
ALTER TABLE wam_group_members DROP COLUMN IF EXISTS effective_rating;
ALTER TABLE wam_group_members DROP COLUMN IF EXISTS joker_points;

-- Remove joker from tournament_participants
ALTER TABLE tournament_participants DROP COLUMN IF EXISTS joker_points;
