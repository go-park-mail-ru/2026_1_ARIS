ALTER TABLE sticker
    ADD COLUMN IF NOT EXISTS media_id BIGINT REFERENCES media(id) ON DELETE SET NULL;

ALTER TYPE reaction_type ADD VALUE IF NOT EXISTS '👍';
ALTER TYPE reaction_type ADD VALUE IF NOT EXISTS '❤️';
ALTER TYPE reaction_type ADD VALUE IF NOT EXISTS '😂';
ALTER TYPE reaction_type ADD VALUE IF NOT EXISTS '😢';
ALTER TYPE reaction_type ADD VALUE IF NOT EXISTS '😡';

ALTER TABLE message
    DROP CONSTRAINT IF EXISTS message_content_check;

ALTER TABLE message
    ADD CONSTRAINT message_content_check CHECK (
        (
            sticker_id IS NOT NULL
            AND message_text IS NULL
        )
        OR
        (
            sticker_id IS NULL
            AND (
                message_text IS NULL
                OR message_text <> ''
            )
        )
    );

CREATE INDEX IF NOT EXISTS comment_post_parent_created_idx
    ON comment (post_id, parent_comment_id, created_at DESC, id DESC)
    WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS comment_parent_created_idx
    ON comment (parent_comment_id, created_at ASC, id ASC)
    WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS message_media_message_order_idx
    ON message_with_media (message_id, sort_order);

CREATE INDEX IF NOT EXISTS post_media_post_order_idx
    ON post_with_media (post_id, sort_order);

CREATE INDEX IF NOT EXISTS sticker_active_pack_order_idx
    ON sticker (pack_id, sort_order)
    WHERE is_active = TRUE;
