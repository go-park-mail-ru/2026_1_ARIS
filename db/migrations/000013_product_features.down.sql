DROP INDEX IF EXISTS sticker_active_pack_order_idx;
DROP INDEX IF EXISTS post_media_post_order_idx;
DROP INDEX IF EXISTS message_media_message_order_idx;
DROP INDEX IF EXISTS comment_parent_created_idx;
DROP INDEX IF EXISTS comment_post_parent_created_idx;

ALTER TABLE message
    DROP CONSTRAINT IF EXISTS message_content_check;

ALTER TABLE message
    ADD CONSTRAINT message_content_check CHECK (
        message_text IS NOT NULL AND message_text <> '' AND sticker_id IS NULL
        OR
        sticker_id IS NOT NULL AND message_text IS NULL
    );

ALTER TABLE sticker
    DROP COLUMN IF EXISTS media_id;
