-- Add completed_count column to tournament_rounds table
ALTER TABLE tournament_rounds ADD COLUMN completed_count INTEGER NOT NULL DEFAULT 0;

-- Initialize completed_count for existing rounds
UPDATE tournament_rounds tr
SET completed_count = (
    SELECT COUNT(*) FROM games g
    WHERE g.round_id = tr.id AND g.status = 'completed'
);
