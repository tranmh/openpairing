-- Revert fide_title to VARCHAR(10)
-- Note: This may fail if there are values longer than 10 characters
ALTER TABLE players ALTER COLUMN fide_title TYPE VARCHAR(10);
