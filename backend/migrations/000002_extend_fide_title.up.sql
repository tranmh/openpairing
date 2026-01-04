-- Extend fide_title column to accommodate multiple arbiter/organizer titles
-- FIDE o_title can contain comma-separated titles like "FI,SI,LSI,AO,IA" (15+ chars)
ALTER TABLE players ALTER COLUMN fide_title TYPE VARCHAR(50);
