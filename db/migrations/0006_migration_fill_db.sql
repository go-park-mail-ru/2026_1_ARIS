INSERT INTO media (uid, media_name, extension, mime_type, description, size) VALUES
('aa000001-0000-0000-0000-000000000001', 'avatar_gleb',        'jpg',  'image/jpeg', 'Аватар Глеба',                          204800),
('aa000001-0000-0000-0000-000000000002', 'avatar_ivan',        'jpg',  'image/jpeg', 'Аватар Ивана',                          189400),
('aa000001-0000-0000-0000-000000000003', 'avatar_anna',        'png',  'image/png',  'Аватар Анны',                           312000),
('aa000001-0000-0000-0000-000000000004', 'avatar_dmitry',      'jpg',  'image/jpeg', 'Аватар Дмитрия',                        95000),
('aa000001-0000-0000-0000-000000000005', 'avatar_maria',       'png',  'image/png',  'Аватар Марии',                          450000),
('aa000001-0000-0000-0000-000000000006', 'post_hiking',        'jpg',  'image/jpeg', 'Фото из похода',                        1048576),
('aa000001-0000-0000-0000-000000000007', 'post_sunset',        'jpg',  'image/jpeg', 'Закат на море',                         2097152),
('aa000001-0000-0000-0000-000000000008', 'post_concert',       'mp4',  'video/mp4',  'Видео с концерта',                      10485760),
('aa000001-0000-0000-0000-000000000009', 'post_code',          'png',  'image/png',  'Скриншот кода',                         180000),
('aa000001-0000-0000-0000-000000000010', 'post_food',          'jpg',  'image/jpeg', 'Ужин в ресторане',                      320000),
('aa000001-0000-0000-0000-000000000011', 'chat_team_avatar',   'png',  'image/png',  'Аватар командного чата',                51200),
('aa000001-0000-0000-0000-000000000012', 'msg_meme',           'jpg',  'image/jpeg', 'Мем в сообщении',                       512000),
('aa000001-0000-0000-0000-000000000013', 'comment_screenshot', 'jpg',  'image/jpeg', 'Скриншот в комментарии',                256000),
('aa000001-0000-0000-0000-000000000014', 'ad_tech_banner',     'png',  'image/png',  'Баннер техно-рекламы',                  307200),
('aa000001-0000-0000-0000-000000000015', 'comm_dev_avatar',    'png',  'image/png',  'Аватар сообщества разработчиков',       102400),
('aa000001-0000-0000-0000-000000000016', 'comm_music_avatar',  'jpg',  'image/jpeg', 'Аватар музыкального сообщества',        88000),
('aa000001-0000-0000-0000-000000000017', 'comm_sport_avatar',  'png',  'image/png',  'Аватар спортивного сообщества',         95000),
('aa000001-0000-0000-0000-000000000018', 'comm_food_avatar',   'jpg',  'image/jpeg', 'Аватар кулинарного сообщества',         110000),
('aa000001-0000-0000-0000-000000000019', 'comm_travel_avatar', 'png',  'image/png',  'Аватар туристического сообщества',      125000),
('aa000001-0000-0000-0000-000000000020', 'ad_food_banner',     'jpg',  'image/jpeg', 'Баннер рекламы кофе',                   280000);

INSERT INTO profile (uid, avatar_id, username) VALUES
('bb000001-0000-0000-0000-000000000001',  1, 'gleb_g'),
('bb000001-0000-0000-0000-000000000002',  2, 'ivan_petrov'),
('bb000001-0000-0000-0000-000000000003',  3, 'anna_k'),
('bb000001-0000-0000-0000-000000000004',  4, 'dima_code'),
('bb000001-0000-0000-0000-000000000005',  5, 'masha_m'),
('bb000001-0000-0000-0000-000000000006', 15, 'dev_community'),
('bb000001-0000-0000-0000-000000000007', 16, 'music_vibes'),
('bb000001-0000-0000-0000-000000000008', 17, 'sport_life'),
('bb000001-0000-0000-0000-000000000009', 18, 'food_lovers'),
('bb000001-0000-0000-0000-000000000010', 19, 'travel_club');

INSERT INTO user_account (uid, email, phone, password_hash) VALUES
('cc000001-0000-0000-0000-000000000001', 'gleb@gmail.com',   '+79991112233', 'bcrypt$2a$10$glebhash'),
('cc000001-0000-0000-0000-000000000002', 'ivan@mail.ru',     '+79992223344', 'bcrypt$2a$10$ivanhash'),
('cc000001-0000-0000-0000-000000000003', 'anna@yandex.ru',   '+79993334455', 'bcrypt$2a$10$annahash'),
('cc000001-0000-0000-0000-000000000004', 'dmitry@gmail.com', '+79994445566', 'bcrypt$2a$10$dimahash'),
('cc000001-0000-0000-0000-000000000005', 'maria@mail.ru',    '+79995556677', 'bcrypt$2a$10$mariahash');

INSERT INTO user_profile (uid, user_account_id, profile_id, first_name, last_name, bio, birthday_date, gender) VALUES
('dd000001-0000-0000-0000-000000000001', 1, 1, 'Глеб',    'Гапонов',  'Backend-разработчик, люблю Go и базы данных', '2001-05-15', 'male'),
('dd000001-0000-0000-0000-000000000002', 2, 2, 'Иван',    'Петров',   'Fullstack dev, React + Node.js',              '1998-11-22', 'male'),
('dd000001-0000-0000-0000-000000000003', 3, 3, 'Анна',    'Козлова',  'UX/UI дизайнер, Figma enthusiast',            '2000-03-08', 'female'),
('dd000001-0000-0000-0000-000000000004', 4, 4, 'Дмитрий', 'Смирнов',  'DevOps, Kubernetes, облачные сервисы',        '1996-07-30', 'male'),
('dd000001-0000-0000-0000-000000000005', 5, 5, 'Мария',   'Новикова', 'Data scientist, Python, ML',                  '1999-12-01', 'female');

INSERT INTO post (uid, post_text, author_id) VALUES
('ee000001-0000-0000-0000-000000000001', 'Сегодня был в походе на Алтае — виды просто огонь! 🏔', 1),
('ee000001-0000-0000-0000-000000000002', 'Опубликовал новый пет-проект на Go. Репо в открытом доступе, жду PR!', 4),
('ee000001-0000-0000-0000-000000000003', 'Лучший закат, который я видела в этом году ☀️ Море не отпускает.', 3),
('ee000001-0000-0000-0000-000000000004', 'Настроил k8s кластер с нуля за выходные. Happy to answer questions.', 4),
('ee000001-0000-0000-0000-000000000005', 'Попробовала новый ресторан на Арбате — однозначно рекомендую!', 5);

INSERT INTO post_with_media (post_id, media_id, sort_order) VALUES
(1, 6,  0),
(2, 9,  0),
(3, 7,  0),
(3, 8,  1),
(4, 9,  0),
(5, 10, 0);

INSERT INTO chat (uid, chat_type, title, avatar_id) VALUES
('ff000001-0000-0000-0000-000000000001', 'personal',  'Глеб и Иван',       NULL),
('ff000001-0000-0000-0000-000000000002', 'personal',  'Анна и Мария',      NULL),
('ff000001-0000-0000-0000-000000000003', 'community', 'Команда ARIS',      11),
('ff000001-0000-0000-0000-000000000004', 'community', 'Go разработчики',   NULL),
('ff000001-0000-0000-0000-000000000005', 'community', 'DevOps чат',        NULL);

INSERT INTO chat_member (uid, chat_id, profile_id, chat_role) VALUES

('b452ec72-d3a4-4b79-bf22-07279e38d9f6', 1, 2, 'member'),

('b452ec72-d3a4-4b79-bf22-07279e38d9f7', 2, 3, 'admin'),
('b452ec72-d3a4-4b79-bf22-07279e38d9f8', 2, 5, 'member'),

('b452ec72-d3a4-4b79-bf22-07279e38d9f9', 3, 1, 'admin'),
('b452ec72-d3a4-4b79-bf22-07279e38d9fa', 3, 2, 'member'),
('b452ec72-d3a4-4b79-bf22-07279e38d9fb', 3, 3, 'member'),
('b452ec72-d3a4-4b79-bf22-07279e38d9fc', 3, 4, 'member'),

('b452ec72-d3a4-4b79-bf22-07279e38d9fd', 4, 4, 'admin'),
('b452ec72-d3a4-4b79-bf22-07279e38d9fe', 4, 1, 'member'),
('b452ec72-d3a4-4b79-bf22-07279e38d9ff', 4, 2, 'member'),

('b452ec72-d3a4-4b79-bf22-07279e38d955', 5, 4, 'admin'),
('b452ec72-d3a4-4b79-bf22-07279e38d922', 5, 5, 'member');

INSERT INTO sticker_pack (uid, title, author_id) VALUES
('c86ca9d6-076d-4ee0-bdf2-dff3086d0285', 'Классические эмоции', 1),
('c86ca9d6-076d-4ee0-bdf2-dff3086d0185', 'Котики',              2),
('c86ca9d6-076d-4ee0-bdf2-dff3086d0485', 'Разработчикам',       4),
('c86ca9d6-076d-4ee0-bdf2-dff3086d0585', 'Аниме реакции',       3),
('c86ca9d6-076d-4ee0-bdf2-dff3086d0685', 'Мемы 2025',           5);

INSERT INTO sticker (uid, size, sort_order, pack_id) VALUES
('3d33c63e-6632-4b83-8110-3ebb211cb078', 45000, 0, 1),
('3d33c63e-6632-4b83-8110-3ebb221cb078', 62000, 0, 2),
('3d33c63e-6632-4b83-8110-3ebb231cb078', 38000, 0, 3),
('3d33c63e-6632-4b83-8110-3ebb241cb078', 54000, 0, 4),
('3d33c63e-6632-4b83-8110-3ebb251cb078', 71000, 0, 5);

INSERT INTO message (uid, message_text, parent_message_id, chat_id, status, sticker_id, author_id) VALUES
('eb9e4a0c-1548-480e-90af-5531e3fc3b1d', 'Привет всем! Как дела с миграциями?',         NULL, 3, 'read',      NULL, 1),
('eb9e3a0c-1548-480e-90af-5532e3fc3b1d', 'Дописал swagger-доки, можно ревьюить',        1,    3, 'read',      NULL, 4),
('eb9e4a0c-1548-480e-90af-5533e3fc3b1d', 'Надо ещё тесты написать, тогда смерджим',     2,    3, 'delivered', NULL, 2),
('eb9e5a0c-1548-480e-90af-5534e3fc3b1d', 'Привет! Что делаешь сегодня вечером?',        NULL, 1, 'read',      NULL, 1),
('eb9e3a0c-1548-480e-90af-5535e3fc3b1d', 'Готовлюсь к дедлайну, всё как обычно 😅',     4,    1, 'sent',      NULL, 2);

INSERT INTO message_with_media (message_id, media_id, sort_order) VALUES
(1, 12, 0),
(2,  9, 0),
(3, 12, 0),
(4,  6, 0),
(5,  7, 0);

INSERT INTO community (uid, title, bio, community_type, profile_id) VALUES
('f664b6cc-d2a4-4fab-8843-dd6d1a133b7a', 'Go Разработчики',  'Сообщество Go-разработчиков России',  'public',   6),
('f664b6cc-d2a4-4fab-8843-dd6d1a233b7a', 'Музыкальный клуб', 'Обсуждаем музыку всех жанров',        'public',   7),
('f664b6cc-d2a4-4fab-8843-dd6d1a333b7a', 'Спорт и здоровье', 'ЗОЖ, тренировки, правильное питание', 'public',   8),
('f664b6cc-d2a4-4fab-8843-dd6d1a433b7a', 'Кулинары',         'Рецепты и советы для гурманов',       'private',  9),
('f664b6cc-d2a4-4fab-8843-dd6d1a533b7a', 'Путешественники',  'Делимся маршрутами и лайфхаками',     'public',   10);

INSERT INTO community_member (uid, profile_id, community_id, community_role) VALUES
('941b1996-340a-451d-a339-e885d82ce320', 1, 1, 'owner'),
('941b2996-340a-451d-a339-e885d82ce320', 2, 1, 'member'),
('941b3996-340a-451d-a339-e885d82ce320', 4, 1, 'admin'),
('941b4996-340a-451d-a339-e885d82ce320', 5, 1, 'member'),
('941b5996-340a-451d-a339-e885d82ce320', 3, 2, 'owner'),
('941b6996-340a-451d-a339-e885d82ce320', 5, 2, 'member'),
('941b7996-340a-451d-a339-e885d82ce320', 4, 3, 'owner'),
('941b8996-340a-451d-a339-e885d82ce320', 1, 3, 'member'),
('941b9996-340a-451d-a339-e885d82ce320', 5, 4, 'owner'),
('941ba996-340a-451d-a339-e885d82ce320', 2, 5, 'owner');

INSERT INTO comment (uid, comment_text, post_id, parent_comment_id, sticker_id, author_id) VALUES
('94edfd61-d5c1-401e-8ba0-d81ae14afed4', 'Потрясающие виды! Где именно снимал?',           1, NULL, NULL, 2),
('94edfd61-d5c1-401e-8ba0-d82ae14afed4', 'Горный Алтай, Мультинские озёра — рекомендую!',  1, 1,    NULL, 1),
('94edfd61-d5c1-401e-8ba0-d83ae14afed4', 'Крутой проект, уже поставил звезду на GitHub!',  2, NULL, NULL, 2),
('94edfd61-d5c1-401e-8ba0-d84ae14afed4', 'Закат просто сказочный, завидую белой завистью', 3, NULL, NULL, 4),
('94edfd61-d5c1-401e-8ba0-d85ae14afed4', 'Запишешь туториал по настройке?',                4, NULL, NULL, 1);

INSERT INTO comment_with_media (comment_id, media_id, sort_order) VALUES
(1, 13, 0),
(2,  6, 0),
(3,  9, 0),
(4,  7, 0),
(5, 13, 0);

INSERT INTO like_record (uid, author_id, post_id) VALUES
('8eaf8e3f-f022-4bfd-b0ae-b8981447e351', 2, 1),
('8eaf8e3f-f022-4bfd-b0ae-b8982447e351', 3, 2),
('8eaf8e3f-f022-4bfd-b0ae-b8983447e351', 4, 3),
('8eaf8e3f-f022-4bfd-b0ae-b8984447e351', 5, 4),
('8eaf8e3f-f022-4bfd-b0ae-b8985447e351', 1, 5),
('8eaf8e3f-f022-4bfd-b0ae-b8986447e351', 3, 1),
('8eaf8e3f-f022-4bfd-b0ae-b8987447e351', 4, 1),
('8eaf8e3f-f022-4bfd-b0ae-b8988447e351', 1, 4),
('8eaf8e3f-f022-4bfd-b0ae-b8989447e351', 2, 4),
('8eaf8e3f-f022-4bfd-b0ae-b898a447e351', 5, 2);

INSERT INTO reaction (uid, message_id, reaction_type, author_id) VALUES
('c2bfafcf-0b3a-44f8-9d7f-0f95198008ba', 1, '\like',    2),
('c2bfafcf-0b3a-44f8-9d7f-0f95298008ba', 1, '\dislike', 3),
('c2bfafcf-0b3a-44f8-9d7f-0f95398008ba', 2, '\anger',   1),
('c2bfafcf-0b3a-44f8-9d7f-0f95498008ba', 3, '\happy',   4),
('c2bfafcf-0b3a-44f8-9d7f-0f95598008ba', 4, '\like',    2);

INSERT INTO friendship (friend1_id, friend2_id, requester_id, status) VALUES
(1, 2, 1, 'accepted'),
(1, 3, 1, 'accepted'),
(2, 3, 2, 'pending'),
(3, 4, 3, 'accepted'),
(4, 5, 4, 'accepted');

INSERT INTO ad (uid, title, description, link, media_id, author_id) VALUES
('6ba7f1e8-a62c-4853-a977-d91a633b2a20', 'Курсы Go с нуля',        'Онлайн-курс по Go для начинающих разработчиков', 'https://go-course.ru',  14, 1),
('6ba7f1e8-a62c-4853-a977-d92a633b2a20', 'Хостинг VDS',            'Быстрые серверы от 299 руб/мес, SLA 99.9%',      'https://vds-host.ru',   NULL, 4),
('6ba7f1e8-a62c-4853-a977-d93a633b2a20', 'MacBook Pro M3',         'Новый MacBook Pro уже в продаже',                'https://apple.ru',      20, 4),
('6ba7f1e8-a62c-4853-a977-d94a633b2a20', 'Figma Pro план',         'Скидка 30% для студентов по промокоду EDU',      'https://figma.com/edu', NULL, 3),
('6ba7f1e8-a62c-4853-a977-d95a633b2a20', 'Кофе для разработчиков', 'Энергетический кофе с бесплатной доставкой',     'https://devcoffee.ru',  20, 5);

INSERT INTO ad_meta (uid, ad_id, meta_key, meta_value) VALUES
('1bfd99cc-ea02-4b05-b975-7151c281b6d7', 1, 'target_audience', 'developers'),
('1bfd99cc-ea02-4b05-b975-7151c282b6d7', 1, 'campaign',        'spring2026'),
('1bfd99cc-ea02-4b05-b975-7151c283b6d7', 2, 'target_audience', 'startups'),
('1bfd99cc-ea02-4b05-b975-7151c284b6d7', 3, 'region',          'ru'),
('1bfd99cc-ea02-4b05-b975-7151c285b6d7', 4, 'target_audience', 'students');