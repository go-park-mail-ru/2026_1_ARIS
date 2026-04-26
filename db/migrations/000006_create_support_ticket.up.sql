CREATE TABLE IF NOT EXISTS support_ticket (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    profile_id BIGINT NOT NULL REFERENCES profile(id) ON DELETE CASCADE,
    login TEXT NOT NULL CHECK (LENGTH(login) >= 1 AND LENGTH(login) <= 255),
    email TEXT NOT NULL CHECK (
        email ~ '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$' AND LENGTH(email) >= 5 AND LENGTH(email) <= 255
    ),
    category INT NOT NULL CHECK (category >= 0 AND category <= 4),
    title TEXT NOT NULL CHECK (LENGTH(title) >= 1 AND LENGTH(title) <= 255),
    description TEXT NOT NULL CHECK (LENGTH(description) >= 1 AND LENGTH(description) <= 5000),
    status INT NOT NULL DEFAULT 0 CHECK (status >= 0 AND status <= 3),
    priority INT NOT NULL DEFAULT 0 CHECK (priority >= 0 AND priority <= 2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_support_ticket_profile_id ON support_ticket(profile_id);
CREATE INDEX IF NOT EXISTS idx_support_ticket_status ON support_ticket(status);
CREATE INDEX IF NOT EXISTS idx_support_ticket_category ON support_ticket(category);
