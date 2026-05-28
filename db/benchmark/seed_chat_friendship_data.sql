SET synchronous_commit = off;

DELETE FROM message
WHERE chat_id IN (SELECT id FROM chat WHERE title = 'perf-chat-history');

DELETE FROM chat_member
WHERE chat_id IN (SELECT id FROM chat WHERE title = 'perf-chat-history');

DELETE FROM chat
WHERE title = 'perf-chat-history';

INSERT INTO chat (uid, chat_type, title, is_active)
VALUES (gen_random_uuid(), 'personal', 'perf-chat-history', TRUE);

INSERT INTO chat_member (uid, chat_id, profile_id, joined_at, chat_role)
SELECT gen_random_uuid(), c.id, p.id, NOW(), 'member'
FROM chat c
JOIN profile p ON p.id IN (1, 2)
WHERE c.title = 'perf-chat-history';

INSERT INTO message (uid, message_text, chat_id, author_id, status, is_active, created_at, updated_at)
SELECT
  gen_random_uuid(),
  'perf message ' || gs,
  c.id,
  CASE WHEN gs % 2 = 0 THEN 1 ELSE 2 END,
  'sent',
  TRUE,
  NOW() - (gs || ' seconds')::interval,
  NOW() - (gs || ' seconds')::interval
FROM generate_series(1, 500000) AS gs
CROSS JOIN (SELECT id FROM chat WHERE title = 'perf-chat-history' ORDER BY id DESC LIMIT 1) c;

TRUNCATE friendship;

INSERT INTO friendship (requester_id, addressee_id, status, created_at, updated_at)
SELECT requester_id, addressee_id, 'accepted', NOW(), NOW()
FROM (
  SELECT a.id AS requester_id, b.id AS addressee_id
  FROM profile a
  JOIN profile b ON b.id > a.id
  WHERE a.id <= 5000 AND b.id <= 5000
  ORDER BY a.id, b.id
  LIMIT 1000000
) pairs;

ANALYZE chat;
ANALYZE chat_member;
ANALYZE message;
ANALYZE friendship;
