-- OpenPairing Schema Rollback Migration
-- Version: 000001
-- Description: Drops all tables created in the up migration

-- Drop triggers first
DROP TRIGGER IF EXISTS update_team_participants_updated_at ON team_tournament_participants;
DROP TRIGGER IF EXISTS update_teams_updated_at ON teams;
DROP TRIGGER IF EXISTS update_participants_updated_at ON tournament_participants;
DROP TRIGGER IF EXISTS update_tournaments_updated_at ON tournaments;
DROP TRIGGER IF EXISTS update_organizations_updated_at ON organizations;
DROP TRIGGER IF EXISTS update_clubs_updated_at ON clubs;
DROP TRIGGER IF EXISTS update_players_updated_at ON players;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS federation_sync_log;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS rating_history;
DROP TABLE IF EXISTS team_tournament_settings;
DROP TABLE IF EXISTS team_match_boards;
DROP TABLE IF EXISTS team_matches;
DROP TABLE IF EXISTS team_lineups;
DROP TABLE IF EXISTS team_tournament_participants;
DROP TABLE IF EXISTS team_members;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS chess_games;
DROP TABLE IF EXISTS forbidden_pairs;
DROP TABLE IF EXISTS games;
DROP TABLE IF EXISTS tournament_rounds;
DROP TABLE IF EXISTS tournament_participants;
DROP TABLE IF EXISTS tournament_sections;
DROP TABLE IF EXISTS tournaments;
DROP TABLE IF EXISTS organization_members;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS players;
DROP TABLE IF EXISTS clubs;
DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;

-- Drop extensions (optional - might be used by other databases)
-- DROP EXTENSION IF EXISTS pg_trgm;
-- DROP EXTENSION IF EXISTS "uuid-ossp";
