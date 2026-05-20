CREATE INDEX IF NOT EXISTS sticker_pack_active_title_trgm_idx
    ON sticker_pack USING GIN (title gin_trgm_ops)
    WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS sticker_pack_active_author_idx
    ON sticker_pack (author_id, created_at DESC, id DESC)
    WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS sticker_media_idx
    ON sticker (media_id)
    WHERE media_id IS NOT NULL AND is_active = TRUE;
