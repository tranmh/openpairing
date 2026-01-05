-- Add ratings locked timestamp to tournaments
ALTER TABLE tournaments ADD COLUMN ratings_locked_at TIMESTAMP WITH TIME ZONE;

-- Add manual rating override to participants
ALTER TABLE tournament_participants ADD COLUMN manual_rating_override INTEGER;

-- Add index for faster lookups on tournaments with locked ratings
CREATE INDEX idx_tournaments_ratings_locked_at ON tournaments(ratings_locked_at) WHERE ratings_locked_at IS NOT NULL;
