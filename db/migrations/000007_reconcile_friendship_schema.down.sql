DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'friendship' AND column_name = 'addressee_id'
    ) THEN
        ALTER TABLE friendship ADD COLUMN addressee_id BIGINT;

        UPDATE friendship
        SET addressee_id = CASE
            WHEN requester_id = friend1_id THEN friend2_id
            ELSE friend1_id
        END;

        ALTER TABLE friendship ALTER COLUMN addressee_id SET NOT NULL;
    END IF;
END $$;

ALTER TABLE friendship DROP CONSTRAINT IF EXISTS friendship_friend1_id_fkey;
ALTER TABLE friendship DROP CONSTRAINT IF EXISTS friendship_friend2_id_fkey;
ALTER TABLE friendship DROP CONSTRAINT IF EXISTS friendship_requester_id_fkey;
ALTER TABLE friendship DROP CONSTRAINT IF EXISTS requester;
ALTER TABLE friendship DROP CONSTRAINT IF EXISTS friendship_order;
ALTER TABLE friendship DROP CONSTRAINT IF EXISTS friendship_pkey;
ALTER TABLE friendship DROP CONSTRAINT IF EXISTS friendship_no_self_reference;

ALTER TABLE friendship ADD CONSTRAINT friendship_no_self_reference CHECK (requester_id <> addressee_id);
ALTER TABLE friendship ADD CONSTRAINT friendship_pkey PRIMARY KEY (requester_id, addressee_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_friend_pair
    ON friendship (LEAST(requester_id, addressee_id), GREATEST(requester_id, addressee_id));

ALTER TABLE friendship DROP COLUMN IF EXISTS is_active;
ALTER TABLE friendship DROP COLUMN IF EXISTS friend1_id;
ALTER TABLE friendship DROP COLUMN IF EXISTS friend2_id;
