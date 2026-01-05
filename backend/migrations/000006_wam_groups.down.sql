-- Drop WAM tables in reverse order

DROP TRIGGER IF EXISTS update_wam_playoff_updated_at ON wam_playoff_matches;
DROP TRIGGER IF EXISTS update_wam_group_members_updated_at ON wam_group_members;
DROP TRIGGER IF EXISTS update_wam_groups_updated_at ON wam_groups;
DROP TRIGGER IF EXISTS update_wam_config_updated_at ON wam_tournament_config;

DROP TABLE IF EXISTS wam_qualifications;
DROP TABLE IF EXISTS wam_playoff_matches;
DROP TABLE IF EXISTS wam_group_games;
DROP TABLE IF EXISTS wam_group_members;
DROP TABLE IF EXISTS wam_groups;
DROP TABLE IF EXISTS wam_tournament_config;
