-- Add bye settings to tournaments (JSONB for flexible configuration)
ALTER TABLE tournaments ADD COLUMN bye_settings JSONB;

-- Add bye tracking fields to tournament participants
ALTER TABLE tournament_participants ADD COLUMN half_point_byes_used INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tournament_participants ADD COLUMN zero_point_byes_used INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tournament_participants ADD COLUMN system_bye_round INTEGER;
ALTER TABLE tournament_participants ADD COLUMN forfeit_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tournament_participants ADD COLUMN consecutive_forfeits INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tournament_participants ADD COLUMN late_entry_round INTEGER;
ALTER TABLE tournament_participants ADD COLUMN withdrawn_round INTEGER;
ALTER TABLE tournament_participants ADD COLUMN withdrawal_reason TEXT;

-- Add index for finding participants with system byes
CREATE INDEX idx_participants_system_bye ON tournament_participants(tournament_id, system_bye_round) WHERE system_bye_round IS NOT NULL;

-- Add index for finding withdrawn participants
CREATE INDEX idx_participants_withdrawn ON tournament_participants(tournament_id, withdrawn_round) WHERE withdrawn_round IS NOT NULL;

-- Add index for finding late entries
CREATE INDEX idx_participants_late_entry ON tournament_participants(tournament_id, late_entry_round) WHERE late_entry_round IS NOT NULL;
