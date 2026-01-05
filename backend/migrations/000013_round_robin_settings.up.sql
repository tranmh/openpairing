-- Add round robin settings column to tournaments table
ALTER TABLE tournaments ADD COLUMN round_robin_settings JSONB DEFAULT NULL;

-- Add comment for documentation
COMMENT ON COLUMN tournaments.round_robin_settings IS 'Round robin specific settings including bye position, fixed player, and seating configuration';
