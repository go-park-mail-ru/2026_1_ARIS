DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'friendship' AND column_name = 'addressee_id'
    ) THEN
        ALTER TABLE friendship ADD COLUMN IF NOT EXISTS friend1_id BIGINT;
        ALTER TABLE friendship ADD COLUMN IF NOT EXISTS friend2_id BIGINT;
        ALTER TABLE friendship ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;

        UPDATE friendship
        SET friend1_id = LEAST(requester_id, addressee_id),
            friend2_id = GREATEST(requester_id, addressee_id)
        WHERE friend1_id IS NULL OR friend2_id IS NULL;

        ALTER TABLE friendship DROP CONSTRAINT IF EXISTS friendship_pkey;
        ALTER TABLE friendship DROP CONSTRAINT IF EXISTS friendship_no_self_reference;
        DROP INDEX IF EXISTS idx_unique_friend_pair;

        ALTER TABLE friendship DROP COLUMN IF EXISTS addressee_id;

        ALTER TABLE friendship ALTER COLUMN friend1_id SET NOT NULL;
        ALTER TABLE friendship ALTER COLUMN friend2_id SET NOT NULL;
    END IF;
END $$;

ALTER TABLE friendship ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'friendship_order'
    ) THEN
        ALTER TABLE friendship
            ADD CONSTRAINT friendship_order CHECK (friend1_id < friend2_id);
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'friendship_no_self_reference'
    ) THEN
        ALTER TABLE friendship
            ADD CONSTRAINT friendship_no_self_reference CHECK (friend1_id <> friend2_id);
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'requester'
    ) THEN
        ALTER TABLE friendship
            ADD CONSTRAINT requester CHECK (requester_id = friend1_id OR requester_id = friend2_id);
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'friendship_pkey'
    ) THEN
        ALTER TABLE friendship
            ADD CONSTRAINT friendship_pkey PRIMARY KEY (friend1_id, friend2_id);
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'friendship_friend1_id_fkey'
    ) THEN
        ALTER TABLE friendship
            ADD CONSTRAINT friendship_friend1_id_fkey
            FOREIGN KEY (friend1_id) REFERENCES profile(id);
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'friendship_friend2_id_fkey'
    ) THEN
        ALTER TABLE friendship
            ADD CONSTRAINT friendship_friend2_id_fkey
            FOREIGN KEY (friend2_id) REFERENCES profile(id);
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'friendship_requester_id_fkey'
    ) THEN
        ALTER TABLE friendship
            ADD CONSTRAINT friendship_requester_id_fkey
            FOREIGN KEY (requester_id) REFERENCES profile(id);
    END IF;
END $$;

