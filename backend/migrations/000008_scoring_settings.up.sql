-- Add scoring settings to tournaments
-- Stores configurable point values for win/draw/loss and Sonneborn-Berger mode
-- Games table continues to store normalized scores (0, 0.5, 1.0)
-- Participant points are stored as display values (converted from normalized)

ALTER TABLE tournaments ADD COLUMN scoring_settings JSONB;

COMMENT ON COLUMN tournaments.scoring_settings IS
    'Configurable scoring: winPoints, drawPoints, lossPoints, sonnebornBergerMode. Games store normalized scores (1/0.5/0) internally.';

-- Create index for tournaments with custom scoring (non-standard)
CREATE INDEX idx_tournaments_custom_scoring ON tournaments(id)
    WHERE scoring_settings IS NOT NULL
    AND scoring_settings->>'preset' != 'standard';
