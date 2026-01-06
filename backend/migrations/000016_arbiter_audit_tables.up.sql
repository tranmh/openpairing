-- Create arbiter_audit_log table
CREATE TABLE IF NOT EXISTS arbiter_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tournament_id UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    arbiter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50),
    entity_id UUID,
    old_value TEXT,
    new_value TEXT,
    reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_arbiter_audit_log_tournament ON arbiter_audit_log(tournament_id);
CREATE INDEX IF NOT EXISTS idx_arbiter_audit_log_arbiter ON arbiter_audit_log(arbiter_id);
CREATE INDEX IF NOT EXISTS idx_arbiter_audit_log_created ON arbiter_audit_log(created_at DESC);

-- Create tournament_announcements table
CREATE TABLE IF NOT EXISTS tournament_announcements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tournament_id UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    priority VARCHAR(50) DEFAULT 'info',
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS idx_announcements_tournament ON tournament_announcements(tournament_id);
CREATE INDEX IF NOT EXISTS idx_announcements_active ON tournament_announcements(tournament_id, is_active);
