CREATE INDEX IF NOT EXISTS post_active_public_feed_idx
    ON post (is_public_demo, created_at DESC, id DESC)
    WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS post_active_author_feed_idx
    ON post (author_id, is_public_demo, created_at DESC, id DESC)
    WHERE is_active = TRUE;
