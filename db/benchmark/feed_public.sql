SELECT id, uid, post_text, author_id, community_id, is_public_demo, allow_comments,
       is_active, created_at, updated_at
FROM post
WHERE is_active = TRUE
  AND is_public_demo = TRUE
ORDER BY created_at DESC, id DESC
LIMIT 50;
