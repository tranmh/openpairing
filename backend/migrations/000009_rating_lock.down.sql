-- Remove index
DROP INDEX IF EXISTS idx_tournaments_ratings_locked_at;

-- Remove manual rating override from participants
ALTER TABLE tournament_participants DROP COLUMN IF EXISTS manual_rating_override;

-- Remove ratings locked timestamp from tournaments
ALTER TABLE tournaments DROP COLUMN IF EXISTS ratings_locked_at;
