DROP INDEX IF EXISTS community_active_username_key;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'community_username_key'
    ) THEN
        ALTER TABLE community
            ADD CONSTRAINT community_username_key UNIQUE (username);
    END IF;
END $$;
