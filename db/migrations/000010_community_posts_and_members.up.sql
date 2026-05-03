ALTER TYPE community_member_role ADD VALUE IF NOT EXISTS 'blocked';

ALTER TABLE community
    ADD COLUMN IF NOT EXISTS cover_media_id BIGINT REFERENCES media(id) ON DELETE SET NULL;

ALTER TABLE post
    ADD COLUMN IF NOT EXISTS community_id BIGINT REFERENCES community(id) ON DELETE SET NULL;

UPDATE community_member
SET community_role = 'moderator'
WHERE community_role = 'manager';

UPDATE post p
SET community_id = c.id
FROM community c
WHERE p.author_id = c.profile_id
  AND p.community_id IS NULL;

CREATE INDEX IF NOT EXISTS community_cover_media_idx
    ON community (cover_media_id)
    WHERE cover_media_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS post_community_created_idx
    ON post (community_id, created_at DESC, id DESC)
    WHERE community_id IS NOT NULL;
