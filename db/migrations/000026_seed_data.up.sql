DO $$
DECLARE
  pwd  CONSTANT TEXT := '$2a$10$OcupBh7XKZN3d7qZWmWMP.rNYvhJobBS4xhdYMd4sPztorUUI3VlO';
  i         INT;
  j         INT;
  liker_idx INT;
  acc_id    BIGINT;
  prof_id   BIGINT;
  media_id  BIGINT;
  cover_id  BIGINT;
  post_id   BIGINT;
  comm_id   BIGINT;
  cprof_id  BIGINT;

  prof_ids     BIGINT[] := '{}';
  cprof_ids    BIGINT[] := '{}';
  comm_ids     BIGINT[] := '{}';
  post_ids     BIGINT[] := '{}';
  post_authors BIGINT[] := '{}';

  fnames TEXT[] := ARRAY[
    'Александр','Михаил','Дмитрий','Иван','Алексей',
    'Андрей','Сергей','Николай','Владимир','Артём',
    'Кирилл','Павел','Максим','Роман','Евгений',
    'Антон','Денис','Илья','Тимур','Даниил',
    'Константин','Вадим','Виктор','Олег','Игорь',
    'Анастасия','Мария','Екатерина','Светлана','Ольга',
    'Наталья','Елена','Юлия','Дарья','Полина',
    'Виктория','Алина','Татьяна','Ирина','Ксения',
    'Валерия','Вера','Надежда','Людмила','Тамара',
    'Нина','Галина','Лариса','Зинаида','Регина'
  ];
  lnames TEXT[] := ARRAY[
    'Иванов','Петров','Сидоров','Козлов','Новиков',
    'Морозов','Соколов','Волков','Попов','Лебедев',
    'Смирнов','Орлов','Федоров','Зайцев','Макаров',
    'Кузнецов','Тихонов','Гусев','Зимин','Борисов',
    'Семёнов','Степанов','Ершов','Беляев','Власов',
    'Иванова','Петрова','Сидорова','Козлова','Новикова',
    'Морозова','Соколова','Волкова','Попова','Лебедева',
    'Смирнова','Орлова','Федорова','Зайцева','Макарова',
    'Кузнецова','Тихонова','Гусева','Зимина','Борисова',
    'Семёнова','Степанова','Ершова','Беляева','Власова'
  ];
  genders TEXT[] := ARRAY[
    'male','male','male','male','male',
    'male','male','male','male','male',
    'male','male','male','male','male',
    'male','male','male','male','male',
    'male','male','male','male','male',
    'female','female','female','female','female',
    'female','female','female','female','female',
    'female','female','female','female','female',
    'female','female','female','female','female',
    'female','female','female','female','female'
  ];
  towns TEXT[] := ARRAY[
    'Москва','Санкт-Петербург','Новосибирск','Екатеринбург','Казань',
    'Нижний Новгород','Челябинск','Самара','Уфа','Ростов-на-Дону',
    'Красноярск','Пермь','Воронеж','Волгоград','Краснодар',
    'Саратов','Тюмень','Тольятти','Ижевск','Барнаул',
    'Ульяновск','Хабаровск','Махачкала','Иркутск','Томск',
    'Москва','Санкт-Петербург','Новосибирск','Екатеринбург','Казань',
    'Нижний Новгород','Челябинск','Самара','Уфа','Ростов-на-Дону',
    'Красноярск','Пермь','Воронеж','Волгоград','Краснодар',
    'Саратов','Тюмень','Тольятти','Ижевск','Барнаул',
    'Ульяновск','Хабаровск','Махачкала','Иркутск','Томск'
  ];
  bios TEXT[] := ARRAY[
    'Люблю технологии и разработку',
    'Студент физического факультета',
    'Программист на Python и Go',
    'Путешествую при каждой возможности',
    'Спорт и здоровый образ жизни',
    'Фотограф-любитель',
    'Читаю книги по истории',
    'Играю на гитаре уже пять лет',
    'Работаю в сфере финансов',
    'Дизайнер интерфейсов',
    'Увлекаюсь астрономией',
    'Кулинарный энтузиаст',
    'Пишу стихи в свободное время',
    'Занимаюсь боксом',
    'Преподаю математику',
    'Стартапер и предприниматель',
    'Фанат научной фантастики',
    'Велосипедист и турист',
    'Учусь на факультете журналистики',
    'Разрабатываю мобильные приложения',
    'Художник и иллюстратор',
    'Работаю в медицине',
    'Люблю настольные игры',
    'Занимаюсь волонтёрством',
    'Интересуюсь экологией и природой'
  ];

  cunames TEXT[] := ARRAY[
    'seedtech','seedsci','seedsport','seedart','seedgame',
    'seedmusic','seedfilm','seedtravel','seedfood','seedphoto'
  ];
  ctitles TEXT[] := ARRAY[
    'Технологии','Наука','Спорт','Искусство','Игры',
    'Музыка','Кино','Путешествия','Кулинария','Фотография'
  ];
  cbios TEXT[] := ARRAY[
    'Обсуждаем новейшие технологии и разработки',
    'Наука вокруг нас: открытия и исследования',
    'Спортивные новости и достижения',
    'Искусство во всех его проявлениях',
    'Игровое сообщество для всех платформ',
    'Музыка без границ и жанров',
    'Кино и сериалы для всех',
    'Впечатления и маршруты путешественников',
    'Рецепты и кулинарные эксперименты',
    'Фотографии и советы по съёмке'
  ];

  ptexts TEXT[] := ARRAY[
    'Отличный день! Много работал над новым проектом.',
    'Только что вернулся из похода в горы. Виды потрясающие!',
    'Нашёл отличную книгу по алгоритмам. Рекомендую всем!',
    'Сегодня приготовил пасту карбонара. Получилось очень вкусно!',
    'Участвовал в хакатоне. Наша команда заняла второе место!',
    'Смотрел новый фильм. Очень понравился сюжет!',
    'Начал изучать машинное обучение. Сложно, но интересно!',
    'Сегодня пробежал десять километров. Личный рекорд!',
    'Встретился с друзьями. Давно так не смеялся!',
    'Закончил читать Войну и Мир. Великая книга!',
    'Новый альбом любимой группы просто огонь!',
    'Сделал ребрендинг своего портфолио. Как вам?',
    'Осваиваю новый язык программирования. Идёт неплохо!',
    'Сегодня закат был просто сказочный. Фото прилагается.',
    'Посетил выставку современного искусства. Вдохновляет!',
    'Первый урок игры на гитаре. Пальцы болят, но счастлив!',
    'Вернулся из командировки. Соскучился по дому.',
    'Новый рецепт смузи: банан, шпинат, имбирь. Советую!',
    'Ночь кодинга позади. Баг наконец пойман!',
    'Наш стартап получил первое финансирование!'
  ];
  ctexts TEXT[] := ARRAY[
    'Обсуждаем новые тренды — присоединяйтесь!',
    'Делимся открытиями этого года.',
    'Итоги сезона — ваши впечатления?',
    'Лучшие работы участников нашего сообщества.',
    'Топ этого месяца по версии сообщества.',
    'Плейлист недели от наших участников.',
    'Обзор лучших релизов этого года.',
    'Фотоотчёт с последней встречи.',
    'Новый конкурс — присоединяйтесь!',
    'Спасибо за активное участие!'
  ];
  cmts TEXT[] := ARRAY[
    'Отличный пост!',
    'Согласен полностью.',
    'Интересно, спасибо!',
    'Не знал об этом.',
    'Красиво!',
    'Подписываюсь под каждым словом.',
    'Тоже так думаю.',
    'Здорово!',
    'Спасибо за информацию.',
    'Очень актуально.',
    'Круто!',
    'Буду иметь в виду.',
    'Как всегда, отлично!',
    'Поддерживаю!',
    'Интересная точка зрения.',
    'Спасибо, познавательно.',
    'Любопытно!',
    'Продолжай в том же духе!',
    'Это вдохновляет.',
    'Замечательно!'
  ];

BEGIN
  IF EXISTS (SELECT 1 FROM user_account WHERE username = 'seed001') THEN
    RAISE NOTICE 'Seed data already present, skipping.';
    RETURN;
  END IF;

  -- ── Phase 1: 50 users ──────────────────────────────────────────────────────
  FOR i IN 1..50 LOOP
    INSERT INTO user_account (uid, email, password_hash, username)
    VALUES (
      gen_random_uuid(),
      'seed' || LPAD(i::TEXT, 3, '0') || '@aris.test',
      pwd,
      'seed' || LPAD(i::TEXT, 3, '0')
    )
    RETURNING id INTO acc_id;

    INSERT INTO profile (uid) VALUES (gen_random_uuid()) RETURNING id INTO prof_id;

    INSERT INTO media (uid, media_name, author_id, extension, mime_type, link, size)
    VALUES (
      gen_random_uuid(),
      'avatar_seed' || LPAD(i::TEXT, 3, '0'),
      prof_id, 'jpg', 'image/jpeg',
      'https://i.pravatar.cc/300?img=' || i,
      0
    )
    RETURNING id INTO media_id;

    UPDATE profile SET avatar_id = media_id WHERE id = prof_id;

    INSERT INTO user_profile (uid, user_account_id, profile_id,
                               first_name, last_name, gender, bio,
                               town, birthday_date)
    VALUES (
      gen_random_uuid(), acc_id, prof_id,
      fnames[i], lnames[i], genders[i]::gender_type,
      bios[((i - 1) % 25) + 1],
      towns[i],
      (DATE '1995-01-01' + ((i * 173 + 31) % 3652) * INTERVAL '1 day')::DATE
    );

    prof_ids := prof_ids || prof_id;
  END LOOP;

  -- ── Phase 2: Friendships (50 × 8 = 400) ──────────────────────────────────
  FOR i IN 1..50 LOOP
    FOR j IN 1..8 LOOP
      INSERT INTO friendship (requester_id, addressee_id, status)
      VALUES (prof_ids[i], prof_ids[((i - 1 + j) % 50) + 1], 'accepted')
      ON CONFLICT (requester_id, addressee_id) DO NOTHING;
    END LOOP;
  END LOOP;

  -- ── Phase 3: 10 communities ───────────────────────────────────────────────
  FOR i IN 1..10 LOOP
    INSERT INTO profile (uid) VALUES (gen_random_uuid()) RETURNING id INTO cprof_id;

    INSERT INTO media (uid, media_name, author_id, extension, mime_type, link, size)
    VALUES (
      gen_random_uuid(), 'comm_avatar_' || cunames[i],
      cprof_id, 'jpg', 'image/jpeg',
      'https://picsum.photos/seed/comav' || i || '/300/300',
      0
    )
    RETURNING id INTO media_id;

    UPDATE profile SET avatar_id = media_id WHERE id = cprof_id;

    INSERT INTO media (uid, media_name, author_id, extension, mime_type, link, size)
    VALUES (
      gen_random_uuid(), 'comm_cover_' || cunames[i],
      cprof_id, 'jpg', 'image/jpeg',
      'https://picsum.photos/seed/comcov' || i || '/1200/400',
      0
    )
    RETURNING id INTO cover_id;

    INSERT INTO community (uid, title, bio, community_type, profile_id, username, cover_media_id)
    VALUES (gen_random_uuid(), ctitles[i], cbios[i], 'public', cprof_id, cunames[i], cover_id)
    RETURNING id INTO comm_id;

    cprof_ids := cprof_ids || cprof_id;
    comm_ids  := comm_ids  || comm_id;

    -- 15 members per community, first one is moderator
    FOR j IN 0..14 LOOP
      INSERT INTO community_member (uid, profile_id, community_id, community_role)
      VALUES (
        gen_random_uuid(),
        prof_ids[((i - 1) * 3 + j) % 50 + 1],
        comm_id,
        CASE WHEN j = 0 THEN 'moderator' ELSE 'member' END::community_member_role
      )
      ON CONFLICT (profile_id, community_id) DO NOTHING;
    END LOOP;
  END LOOP;

  -- ── Phase 4: Personal posts with images (50) ─────────────────────────────
  FOR i IN 1..50 LOOP
    INSERT INTO media (uid, media_name, author_id, extension, mime_type, link, size)
    VALUES (
      gen_random_uuid(), 'post_img_' || i,
      prof_ids[i], 'jpg', 'image/jpeg',
      'https://picsum.photos/seed/post' || i || '/800/600',
      0
    )
    RETURNING id INTO media_id;

    INSERT INTO post (uid, post_text, author_id, is_public_demo)
    VALUES (gen_random_uuid(), ptexts[((i - 1) % 20) + 1], prof_ids[i], FALSE)
    RETURNING id INTO post_id;

    INSERT INTO post_with_media (post_id, media_id, sort_order)
    VALUES (post_id, media_id, 0);

    post_ids     := post_ids     || post_id;
    post_authors := post_authors || prof_ids[i];
  END LOOP;

  -- Text-only posts for users 1-30
  FOR i IN 1..30 LOOP
    INSERT INTO post (uid, post_text, author_id, is_public_demo)
    VALUES (gen_random_uuid(), ptexts[((i + 9) % 20) + 1], prof_ids[i], FALSE)
    RETURNING id INTO post_id;

    post_ids     := post_ids     || post_id;
    post_authors := post_authors || prof_ids[i];
  END LOOP;

  -- Community posts (3 per community = 30)
  FOR i IN 1..10 LOOP
    FOR j IN 1..3 LOOP
      INSERT INTO media (uid, media_name, author_id, extension, mime_type, link, size)
      VALUES (
        gen_random_uuid(),
        'comm_post_' || i || '_' || j,
        cprof_ids[i], 'jpg', 'image/jpeg',
        'https://picsum.photos/seed/cpost' || (i * 10 + j) || '/800/600',
        0
      )
      RETURNING id INTO media_id;

      INSERT INTO post (uid, post_text, author_id, community_id, is_public_demo)
      VALUES (
        gen_random_uuid(),
        ctexts[((i + j - 2) % 10) + 1],
        cprof_ids[i], comm_ids[i], FALSE
      )
      RETURNING id INTO post_id;

      INSERT INTO post_with_media (post_id, media_id, sort_order)
      VALUES (post_id, media_id, 0);

      post_ids     := post_ids     || post_id;
      post_authors := post_authors || cprof_ids[i];
    END LOOP;
  END LOOP;

  -- ── Phase 5: Likes (≈3 per post) ─────────────────────────────────────────
  FOR i IN 1..array_length(post_ids, 1) LOOP
    FOR j IN 1..3 LOOP
      liker_idx := ((i * 7 + j * 13) % 50) + 1;
      IF prof_ids[liker_idx] <> post_authors[i] THEN
        INSERT INTO like_record (uid, post_id, author_id)
        VALUES (gen_random_uuid(), post_ids[i], prof_ids[liker_idx])
        ON CONFLICT DO NOTHING;
      END IF;
    END LOOP;
  END LOOP;

  -- ── Phase 6: Comments (1 per post) ───────────────────────────────────────
  FOR i IN 1..array_length(post_ids, 1) LOOP
    liker_idx := ((i * 11 + 5) % 50) + 1;
    IF prof_ids[liker_idx] <> post_authors[i] THEN
      INSERT INTO comment (uid, comment_text, post_id, author_id)
      VALUES (gen_random_uuid(), cmts[((i - 1) % 20) + 1], post_ids[i], prof_ids[liker_idx]);
    END IF;
  END LOOP;

  -- ── Phase 7: 40 game questions ────────────────────────────────────────────
  -- slug column was dropped by migration 000024, so it is excluded here
  INSERT INTO game_question (uid, game_type, question_text, correct_answer, answer_unit)
  VALUES
    (gen_random_uuid(),'number_duel','В каком году началась Вторая мировая война?',1939,'год'),
    (gen_random_uuid(),'number_duel','В каком году Юрий Гагарин совершил первый космический полёт?',1961,'год'),
    (gen_random_uuid(),'number_duel','В каком году Колумб достиг берегов Америки?',1492,'год'),
    (gen_random_uuid(),'number_duel','В каком году произошла битва при Ватерлоо?',1815,'год'),
    (gen_random_uuid(),'number_duel','Сколько лет фактически длилась Столетняя война?',116,'лет'),
    (gen_random_uuid(),'number_duel','В каком году была принята Конституция США?',1787,'год'),
    (gen_random_uuid(),'number_duel','В каком году прекратил существование СССР?',1991,'год'),
    (gen_random_uuid(),'number_duel','В каком году Пётр I основал Санкт-Петербург?',1703,'год'),
    (gen_random_uuid(),'number_duel','Скорость света в вакууме (км/с, округлённо)?',299792,'км/с'),
    (gen_random_uuid(),'number_duel','Сколько планет в Солнечной системе?',8,'планет'),
    (gen_random_uuid(),'number_duel','Атомный номер водорода в таблице Менделеева?',1,'номер'),
    (gen_random_uuid(),'number_duel','В каком году был открыт пенициллин?',1928,'год'),
    (gen_random_uuid(),'number_duel','Температура кипения воды при нормальном давлении (°C)?',100,'°C'),
    (gen_random_uuid(),'number_duel','Сколько костей в скелете взрослого человека?',206,'костей'),
    (gen_random_uuid(),'number_duel','Молярная масса углекислого газа (г/моль)?',44,'г/моль'),
    (gen_random_uuid(),'number_duel','Средний диаметр Земли (км)?',12742,'км'),
    (gen_random_uuid(),'number_duel','Сколько простых чисел от 1 до 50 включительно?',15,'чисел'),
    (gen_random_uuid(),'number_duel','Квадратный корень из 144?',12,NULL),
    (gen_random_uuid(),'number_duel','Сколько нулей содержит число один миллиард?',9,'нулей'),
    (gen_random_uuid(),'number_duel','Сколько градусов в полном круге?',360,'градусов'),
    (gen_random_uuid(),'number_duel','Какое седьмое число в последовательности Фибоначчи?',13,NULL),
    (gen_random_uuid(),'number_duel','Длина реки Нил (км)?',6853,'км'),
    (gen_random_uuid(),'number_duel','В каком году был открыт Суэцкий канал?',1869,'год'),
    (gen_random_uuid(),'number_duel','Высота горы Эверест над уровнем моря (метров)?',8849,'метров'),
    (gen_random_uuid(),'number_duel','Сколько официально признанных стран в Африке?',54,'стран'),
    (gen_random_uuid(),'number_duel','Площадь России в миллионах кв. км (округлённо)?',17,'млн км²'),
    (gen_random_uuid(),'number_duel','В каком году официально распались The Beatles?',1970,'год'),
    (gen_random_uuid(),'number_duel','Сколько струн у классической гитары?',6,'струн'),
    (gen_random_uuid(),'number_duel','В каком году написана Лунная соната Бетховена?',1801,'год'),
    (gen_random_uuid(),'number_duel','Сколько нот в одной октаве?',7,'нот'),
    (gen_random_uuid(),'number_duel','Сколько игроков от команды на площадке в баскетболе?',5,'игроков'),
    (gen_random_uuid(),'number_duel','В каком году основан ФК Реал Мадрид?',1902,'год'),
    (gen_random_uuid(),'number_duel','В каком году прошли первые современные Олимпийские игры?',1896,'год'),
    (gen_random_uuid(),'number_duel','Сколько геймов нужно выиграть для победы в сете (минимум)?',6,'геймов'),
    (gen_random_uuid(),'number_duel','Сколько минут длится стандартный футбольный матч?',90,'минут'),
    (gen_random_uuid(),'number_duel','Длина дистанции марафона (км, округлённо)?',42,'км'),
    (gen_random_uuid(),'number_duel','В каком году создан язык программирования Python?',1991,'год'),
    (gen_random_uuid(),'number_duel','Сколько бит в одном байте?',8,'бит'),
    (gen_random_uuid(),'number_duel','В каком году выпущен первый микропроцессор Intel 4004?',1971,'год'),
    (gen_random_uuid(),'number_duel','Количество пикселей по горизонтали в Full HD?',1920,'пикселей');

  -- ── Phase 8: Search outbox ────────────────────────────────────────────────
  FOR i IN 1..50 LOOP
    INSERT INTO search_outbox (entity_type, entity_id, operation)
    VALUES ('user', prof_ids[i], 'upsert');
  END LOOP;

  FOR i IN 1..10 LOOP
    INSERT INTO search_outbox (entity_type, entity_id, operation)
    VALUES ('community', comm_ids[i], 'upsert');
  END LOOP;

  FOR i IN 1..array_length(post_ids, 1) LOOP
    INSERT INTO search_outbox (entity_type, entity_id, operation)
    VALUES ('post', post_ids[i], 'upsert');
  END LOOP;

END $$;
