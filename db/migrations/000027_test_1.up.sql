DO $$
DECLARE
  acc_id   BIGINT;
  prof_id  BIGINT;
  media_id BIGINT;
  post_id  BIGINT;
BEGIN
  IF EXISTS (SELECT 1 FROM user_account WHERE username = 'deploy_test_27') THEN
    RAISE NOTICE 'Migration 27 seed already present, skipping.';
    RETURN;
  END IF;

  INSERT INTO profile (uid) VALUES (gen_random_uuid()) RETURNING id INTO prof_id;

  INSERT INTO media (uid, media_name, author_id, extension, mime_type, link, size)
  VALUES (
    gen_random_uuid(), 'deploy_test_27_avatar',
    prof_id, 'jpg', 'image/jpeg',
    'https://i.pravatar.cc/300?img=60',
    0
  )
  RETURNING id INTO media_id;

  UPDATE profile SET avatar_id = media_id WHERE id = prof_id;

  INSERT INTO user_account (uid, email, password_hash, username)
  VALUES (
    gen_random_uuid(),
    'deploy_test_27@aris.test',
    '$2a$10$OcupBh7XKZN3d7qZWmWMP.rNYvhJobBS4xhdYMd4sPztorUUI3VlO',
    'deploy_test_27'
  )
  RETURNING id INTO acc_id;

  INSERT INTO user_profile (uid, user_account_id, profile_id, first_name, last_name, gender, bio, town, birthday_date)
  VALUES (
    gen_random_uuid(), acc_id, prof_id,
    'Deploy', 'Test',
    'male',
    'Тестовый пользователь для проверки деплоя миграции 27',
    'Москва',
    '2000-01-27'
  );

  INSERT INTO post (uid, post_text, author_id, is_public_demo)
  VALUES (
    gen_random_uuid(),
    'Тестовый пост от миграции 27. Если вы это видите — деплой прошёл успешно.',
    prof_id,
    FALSE
  )
  RETURNING id INTO post_id;

  INSERT INTO search_outbox (entity_type, entity_id, operation)
  VALUES ('user', prof_id, 'upsert');

  INSERT INTO search_outbox (entity_type, entity_id, operation)
  VALUES ('post', post_id, 'upsert');
END $$;
