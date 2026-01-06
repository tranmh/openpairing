-- Remove completed_count column from tournament_rounds table
ALTER TABLE tournament_rounds DROP COLUMN IF EXISTS completed_count;
