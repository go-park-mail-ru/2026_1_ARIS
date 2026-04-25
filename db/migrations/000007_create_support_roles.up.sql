CREATE TABLE IF NOT EXISTS support_profile_role (
    profile_id BIGINT PRIMARY KEY REFERENCES profile(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('admin', 'support'))
);

CREATE INDEX IF NOT EXISTS idx_support_profile_role_role ON support_profile_role(role);
