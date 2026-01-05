-- Remove scoring settings from tournaments
DROP INDEX IF EXISTS idx_tournaments_custom_scoring;
ALTER TABLE tournaments DROP COLUMN IF EXISTS scoring_settings;
