-- Create audit_logs table for comprehensive audit trail
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID,
    old_value JSONB,
    new_value JSONB,
    ip_address VARCHAR(45),
    user_agent TEXT,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for quick lookups by user_id
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);

-- Index for entity lookups
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);

-- Index for action filtering
CREATE INDEX idx_audit_logs_action ON audit_logs(action);

-- Index for date range queries
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);

-- Composite index for common query patterns
CREATE INDEX idx_audit_logs_user_action ON audit_logs(user_id, action, created_at DESC);

-- Comment on table
COMMENT ON TABLE audit_logs IS 'Stores audit trail of all significant actions in the system';
