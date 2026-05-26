DO $$
DECLARE
  acc_id  BIGINT;
  prof_id BIGINT;
BEGIN
  SELECT ua.id, up.profile_id
  INTO acc_id, prof_id
  FROM user_account ua
  JOIN user_profile up ON up.user_account_id = ua.id
  WHERE ua.username = 'deploy_test_27'
  LIMIT 1;

  IF acc_id IS NULL THEN
    RAISE NOTICE 'Migration 27 seed not found, nothing to remove.';
    RETURN;
  END IF;

  DELETE FROM search_outbox WHERE entity_type = 'post'
    AND entity_id IN (SELECT id FROM post WHERE author_id = prof_id);
  DELETE FROM search_outbox WHERE entity_type = 'user' AND entity_id = prof_id;

  DELETE FROM post WHERE author_id = prof_id;
  DELETE FROM user_profile WHERE user_account_id = acc_id;
  DELETE FROM user_account WHERE id = acc_id;
  DELETE FROM media WHERE author_id = prof_id;
  DELETE FROM profile WHERE id = prof_id;
END $$;
