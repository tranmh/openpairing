-- Drop push_subscriptions table
DROP INDEX IF EXISTS idx_push_subscriptions_endpoint;
DROP INDEX IF EXISTS idx_push_subscriptions_user_id;
DROP TABLE IF EXISTS push_subscriptions;
