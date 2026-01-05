-- Drop tournament waitlist
DROP INDEX IF EXISTS idx_waitlist_player;
DROP INDEX IF EXISTS idx_waitlist_position;
DROP INDEX IF EXISTS idx_waitlist_tournament;
DROP TABLE IF EXISTS tournament_waitlist;

-- Drop email verification tokens
DROP INDEX IF EXISTS idx_email_verification_tokens_hash;
DROP INDEX IF EXISTS idx_email_verification_tokens_user;
DROP TABLE IF EXISTS email_verification_tokens;
