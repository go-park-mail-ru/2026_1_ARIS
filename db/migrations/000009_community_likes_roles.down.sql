DROP INDEX IF EXISTS community_member_active_idx;
DROP INDEX IF EXISTS community_profile_active_idx;
DROP INDEX IF EXISTS like_record_comment_author_unique;
DROP INDEX IF EXISTS like_record_post_author_unique;

-- PostgreSQL enum values cannot be removed safely in a generic down migration.
