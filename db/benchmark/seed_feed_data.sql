SET synchronous_commit = off;

INSERT INTO profile (uid)
SELECT gen_random_uuid()
FROM generate_series(1, 5000);

INSERT INTO post (uid, post_text, author_id, is_public_demo, allow_comments, is_active, created_at, updated_at)
SELECT
  gen_random_uuid(),
  'perf post ' || gs,
  1 + (gs % 5000),
  (gs % 2 = 0),
  TRUE,
  TRUE,
  NOW() - (gs || ' seconds')::interval,
  NOW() - (gs || ' seconds')::interval
FROM generate_series(1, 500000) AS gs;

ANALYZE profile;
ANALYZE post;
