ALTER TYPE community_member_role ADD VALUE IF NOT EXISTS 'manager';

CREATE UNIQUE INDEX IF NOT EXISTS like_record_post_author_unique
    ON like_record (post_id, author_id)
    WHERE post_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS like_record_comment_author_unique
    ON like_record (comment_id, author_id)
    WHERE comment_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS community_profile_active_idx
    ON community (profile_id)
    WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS community_member_active_idx
    ON community_member (community_id, profile_id)
    WHERE is_active = TRUE;
