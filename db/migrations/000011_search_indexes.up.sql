CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS user_account_username_trgm_idx
    ON user_account USING GIN (username gin_trgm_ops)
    WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS user_profile_first_name_trgm_idx
    ON user_profile USING GIN (first_name gin_trgm_ops)
    WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS user_profile_last_name_trgm_idx
    ON user_profile USING GIN (last_name gin_trgm_ops)
    WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS user_profile_full_name_trgm_idx
    ON user_profile USING GIN ((first_name || ' ' || last_name) gin_trgm_ops)
    WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS community_public_username_trgm_idx
    ON community USING GIN (username gin_trgm_ops)
    WHERE is_active = TRUE AND community_type = 'public';

CREATE INDEX IF NOT EXISTS community_public_title_trgm_idx
    ON community USING GIN (title gin_trgm_ops)
    WHERE is_active = TRUE AND community_type = 'public';

CREATE INDEX IF NOT EXISTS community_public_bio_trgm_idx
    ON community USING GIN (bio gin_trgm_ops)
    WHERE is_active = TRUE AND community_type = 'public' AND bio IS NOT NULL;
