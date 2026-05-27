DO $$
DECLARE
  seed_profs BIGINT[];
  comm_profs BIGINT[];
  all_profs  BIGINT[];
BEGIN
  SELECT array_agg(up.profile_id) INTO seed_profs
  FROM user_account ua
  JOIN user_profile up ON up.user_account_id = ua.id
  WHERE ua.username LIKE 'seed%';

  SELECT array_agg(profile_id) INTO comm_profs
  FROM community
  WHERE username LIKE 'seed%';

  IF seed_profs IS NULL AND comm_profs IS NULL THEN
    RAISE NOTICE 'No seed data found, skipping.';
    RETURN;
  END IF;

  seed_profs := COALESCE(seed_profs, '{}');
  comm_profs := COALESCE(comm_profs, '{}');
  all_profs  := seed_profs || comm_profs;

  -- Search outbox
  DELETE FROM search_outbox
  WHERE (entity_type = 'user'      AND entity_id = ANY(seed_profs))
     OR (entity_type = 'community' AND entity_id IN (SELECT id FROM community WHERE username LIKE 'seed%'))
     OR (entity_type = 'post'      AND entity_id IN (SELECT id FROM post WHERE author_id = ANY(all_profs)));

  -- Game questions added by this migration (slug column dropped in 000024, delete by question_text)
  DELETE FROM game_question WHERE question_text IN (
    'В каком году началась Вторая мировая война?',
    'В каком году Юрий Гагарин совершил первый космический полёт?',
    'В каком году Колумб достиг берегов Америки?',
    'В каком году произошла битва при Ватерлоо?',
    'Сколько лет фактически длилась Столетняя война?',
    'В каком году была принята Конституция США?',
    'В каком году прекратил существование СССР?',
    'В каком году Пётр I основал Санкт-Петербург?',
    'Скорость света в вакууме (км/с, округлённо)?',
    'Сколько планет в Солнечной системе?',
    'Атомный номер водорода в таблице Менделеева?',
    'В каком году был открыт пенициллин?',
    'Температура кипения воды при нормальном давлении (°C)?',
    'Сколько костей в скелете взрослого человека?',
    'Молярная масса углекислого газа (г/моль)?',
    'Средний диаметр Земли (км)?',
    'Сколько простых чисел от 1 до 50 включительно?',
    'Квадратный корень из 144?',
    'Сколько нулей содержит число один миллиард?',
    'Сколько градусов в полном круге?',
    'Какое седьмое число в последовательности Фибоначчи?',
    'Длина реки Нил (км)?',
    'В каком году был открыт Суэцкий канал?',
    'Высота горы Эверест над уровнем моря (метров)?',
    'Сколько официально признанных стран в Африке?',
    'Площадь России в миллионах кв. км (округлённо)?',
    'В каком году официально распались The Beatles?',
    'Сколько струн у классической гитары?',
    'В каком году написана Лунная соната Бетховена?',
    'Сколько нот в одной октаве?',
    'Сколько игроков от команды на площадке в баскетболе?',
    'В каком году основан ФК Реал Мадрид?',
    'В каком году прошли первые современные Олимпийские игры?',
    'Сколько геймов нужно выиграть для победы в сете (минимум)?',
    'Сколько минут длится стандартный футбольный матч?',
    'Длина дистанции марафона (км, округлённо)?',
    'В каком году создан язык программирования Python?',
    'Сколько бит в одном байте?',
    'В каком году выпущен первый микропроцессор Intel 4004?',
    'Количество пикселей по горизонтали в Full HD?'
  );

  -- Likes on posts by seed/community profiles (FK is RESTRICT, must go before posts)
  DELETE FROM like_record
  WHERE post_id IN (SELECT id FROM post WHERE author_id = ANY(all_profs));

  -- Comments cascade-delete with post, but likes on comments need removing first
  DELETE FROM like_record
  WHERE comment_id IN (
    SELECT c.id FROM comment c
    JOIN post p ON p.id = c.post_id
    WHERE p.author_id = ANY(all_profs)
  );

  -- Remove posts (CASCADE removes post_with_media, comments)
  DELETE FROM post WHERE author_id = ANY(all_profs);

  -- Remove community members for seed users (communities CASCADE the rest)
  DELETE FROM community_member WHERE profile_id = ANY(seed_profs);

  -- Remove communities (CASCADE removes remaining community_member rows)
  DELETE FROM community WHERE username LIKE 'seed%';

  -- Remove friendships involving seed users
  DELETE FROM friendship
  WHERE requester_id = ANY(seed_profs) OR addressee_id = ANY(seed_profs);

  -- Remove media owned by seed/community profiles
  UPDATE profile SET avatar_id = NULL WHERE id = ANY(all_profs);
  DELETE FROM media WHERE author_id = ANY(all_profs);

  -- Remove user accounts (CASCADE removes user_profile, user_settings)
  DELETE FROM user_account WHERE username LIKE 'seed%';

  -- Remove profiles
  DELETE FROM profile WHERE id = ANY(all_profs);
END $$;
