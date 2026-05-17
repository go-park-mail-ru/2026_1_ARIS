ALTER TABLE community
    DROP CONSTRAINT IF EXISTS community_username_key;

CREATE UNIQUE INDEX IF NOT EXISTS community_active_username_key
    ON community (username)
    WHERE is_active = TRUE;
