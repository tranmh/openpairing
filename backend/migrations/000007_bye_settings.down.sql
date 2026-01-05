-- Remove indexes
DROP INDEX IF EXISTS idx_participants_late_entry;
DROP INDEX IF EXISTS idx_participants_withdrawn;
DROP INDEX IF EXISTS idx_participants_system_bye;

-- Remove bye tracking fields from participants
ALTER TABLE tournament_participants DROP COLUMN IF EXISTS withdrawal_reason;
ALTER TABLE tournament_participants DROP COLUMN IF EXISTS withdrawn_round;
ALTER TABLE tournament_participants DROP COLUMN IF EXISTS late_entry_round;
ALTER TABLE tournament_participants DROP COLUMN IF EXISTS consecutive_forfeits;
ALTER TABLE tournament_participants DROP COLUMN IF EXISTS forfeit_count;
ALTER TABLE tournament_participants DROP COLUMN IF EXISTS system_bye_round;
ALTER TABLE tournament_participants DROP COLUMN IF EXISTS zero_point_byes_used;
ALTER TABLE tournament_participants DROP COLUMN IF EXISTS half_point_byes_used;

-- Remove bye settings from tournaments
ALTER TABLE tournaments DROP COLUMN IF EXISTS bye_settings;
