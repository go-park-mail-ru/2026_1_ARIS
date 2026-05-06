DROP INDEX IF EXISTS post_community_created_idx;
DROP INDEX IF EXISTS community_cover_media_idx;

ALTER TABLE post
    DROP COLUMN IF EXISTS community_id;

ALTER TABLE community
    DROP COLUMN IF EXISTS cover_media_id;

-- PostgreSQL enum values are not removed in down migrations.
