-- Revert Registration Settings

-- Drop trigger and function
DROP TRIGGER IF EXISTS trigger_guest_registration_updated_at ON tournament_guest_registrations;
DROP FUNCTION IF EXISTS update_guest_registration_updated_at();

-- Drop invite tokens table
DROP TABLE IF EXISTS tournament_invite_tokens;

-- Remove check-in fields from tournament_participants
ALTER TABLE tournament_participants
DROP COLUMN IF EXISTS pre_registered_at,
DROP COLUMN IF EXISTS approved_at,
DROP COLUMN IF EXISTS approved_by,
DROP COLUMN IF EXISTS checked_in_at,
DROP COLUMN IF EXISTS checked_in_by,
DROP COLUMN IF EXISTS check_in_method,
DROP COLUMN IF EXISTS check_in_code,
DROP COLUMN IF EXISTS registration_source;

-- Drop guest registrations table
DROP TABLE IF EXISTS tournament_guest_registrations;

-- Remove registration_settings from tournaments
DROP INDEX IF EXISTS idx_tournaments_registration_settings;
ALTER TABLE tournaments DROP COLUMN IF EXISTS registration_settings;
