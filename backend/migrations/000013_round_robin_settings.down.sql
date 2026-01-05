-- Remove round robin settings column from tournaments table
ALTER TABLE tournaments DROP COLUMN IF EXISTS round_robin_settings;
