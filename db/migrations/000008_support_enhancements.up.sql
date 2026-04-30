ALTER TABLE support_profile_role
    DROP CONSTRAINT IF EXISTS support_profile_role_role_check;

UPDATE support_profile_role
SET role = 'support_l1'
WHERE role = 'support';

ALTER TABLE support_profile_role
    ADD CONSTRAINT support_profile_role_role_check
    CHECK (role IN ('support_l1', 'support_l2', 'admin'));

ALTER TABLE support_ticket
    ADD COLUMN IF NOT EXISTS uid UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    ADD COLUMN IF NOT EXISTS line INT NOT NULL DEFAULT 1 CHECK (line IN (1, 2)),
    ADD COLUMN IF NOT EXISTS assigned_agent_id BIGINT REFERENCES profile(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS rating INT CHECK (rating IS NULL OR rating BETWEEN 1 AND 5);

CREATE INDEX IF NOT EXISTS idx_support_ticket_line ON support_ticket(line);
CREATE INDEX IF NOT EXISTS idx_support_ticket_assigned_agent_id ON support_ticket(assigned_agent_id);
CREATE INDEX IF NOT EXISTS idx_support_ticket_rating ON support_ticket(rating);

CREATE TABLE IF NOT EXISTS ticket_with_media (
    ticket_id BIGINT NOT NULL REFERENCES support_ticket(id) ON DELETE CASCADE,
    media_id BIGINT NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    sort_order INT NOT NULL DEFAULT 0 CHECK (sort_order >= 0 AND sort_order <= 10),
    CONSTRAINT unique_order_per_ticket UNIQUE (ticket_id, sort_order),
    PRIMARY KEY (ticket_id, media_id)
);

CREATE INDEX IF NOT EXISTS idx_ticket_with_media_ticket_id ON ticket_with_media(ticket_id);

CREATE TABLE IF NOT EXISTS support_ticket_message (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    ticket_id BIGINT NOT NULL REFERENCES support_ticket(id) ON DELETE CASCADE,
    text TEXT NOT NULL CHECK (LENGTH(text) >= 1 AND LENGTH(text) <= 5000),
    author_id BIGINT NOT NULL REFERENCES profile(id) ON DELETE CASCADE,
    author_role TEXT NOT NULL CHECK (author_role IN ('user', 'support_l1', 'support_l2', 'admin')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_support_ticket_message_ticket_id ON support_ticket_message(ticket_id, created_at);
