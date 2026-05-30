ALTER TABLE game_room
    ADD COLUMN IF NOT EXISTS is_public_lobby BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS game_public_participant (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID NOT NULL,
    room_id BIGINT NOT NULL REFERENCES game_room(id) ON DELETE CASCADE,
    profile_id BIGINT NOT NULL UNIQUE REFERENCES profile(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE CHECK (LENGTH(token_hash) = 64),
    first_name TEXT NOT NULL CHECK (
        LENGTH(BTRIM(first_name)) >= 1
        AND LENGTH(first_name) <= 12
        AND first_name ~ '^[A-Za-zА-Яа-яЁё-]+$'
    ),
    last_name TEXT NOT NULL CHECK (
        LENGTH(BTRIM(last_name)) >= 1
        AND LENGTH(last_name) <= 12
        AND last_name ~ '^[A-Za-zА-Яа-яЁё-]+$'
    ),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT game_public_participant_room_profile_unique UNIQUE (room_id, profile_id)
);

CREATE TABLE IF NOT EXISTS game_public_lobby_question (
    position INT PRIMARY KEY CHECK (position >= 1),
    question_id BIGINT NOT NULL UNIQUE REFERENCES game_question(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_game_public_participant_room
    ON game_public_participant(room_id)
    WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS idx_game_room_public_waiting
    ON game_room(is_public_lobby, status)
    WHERE is_active = TRUE;

WITH seed(position, uid, game_type, question_text_ru, question_text_en, correct_answer) AS (
    VALUES
        (1, '22222222-2222-4222-8222-222222222201'::uuid, 'number_duel',
         'Сколько планет в Солнечной системе?',
         'How many planets are there in the Solar System?',
         8::DOUBLE PRECISION),
        (2, '22222222-2222-4222-8222-222222222202'::uuid, 'number_duel',
         'В каком году человек впервые высадился на Луне?',
         'In what year did humans first land on the Moon?',
         1969::DOUBLE PRECISION),
        (3, '22222222-2222-4222-8222-222222222203'::uuid, 'number_duel',
         'Сколько колец на олимпийском флаге?',
         'How many rings are on the Olympic flag?',
         5::DOUBLE PRECISION),
        (4, '22222222-2222-4222-8222-222222222204'::uuid, 'number_duel',
         'Сколько клеток на шахматной доске?',
         'How many squares are there on a chessboard?',
         64::DOUBLE PRECISION),
        (5, '22222222-2222-4222-8222-222222222205'::uuid, 'number_duel',
         'В каком году Apple представила первый iPhone?',
         'In what year did Apple introduce the first iPhone?',
         2007::DOUBLE PRECISION),
        (6, '22222222-2222-4222-8222-222222222206'::uuid, 'number_duel',
         'Сколько химических элементов официально входит в периодическую таблицу?',
         'How many chemical elements are officially in the periodic table?',
         118::DOUBLE PRECISION),
        (7, '22222222-2222-4222-8222-222222222207'::uuid, 'number_duel',
         'В каком году затонул Титаник?',
         'In what year did the Titanic sink?',
         1912::DOUBLE PRECISION),
        (8, '22222222-2222-4222-8222-222222222208'::uuid, 'number_duel',
         'Сколько фильмов входит в основную серию о Гарри Поттере?',
         'How many films are in the main Harry Potter series?',
         8::DOUBLE PRECISION)
),
existing AS (
    SELECT DISTINCT ON (s.position)
        s.position,
        q.id AS question_id
    FROM seed s
    JOIN game_question q
      ON q.game_type = s.game_type
     AND q.question_text_ru = s.question_text_ru
     AND q.correct_answer = s.correct_answer
    ORDER BY s.position, q.id
),
inserted AS (
    INSERT INTO game_question (uid, game_type, question_text, question_text_ru, question_text_en, correct_answer, is_active)
    SELECT s.uid, s.game_type, s.question_text_ru, s.question_text_ru, s.question_text_en, s.correct_answer, TRUE
    FROM seed s
    WHERE NOT EXISTS (
        SELECT 1
        FROM existing e
        WHERE e.position = s.position
    )
    RETURNING id, uid
),
resolved AS (
    SELECT position, question_id
    FROM existing
    UNION ALL
    SELECT s.position, i.id
    FROM seed s
    JOIN inserted i ON i.uid = s.uid
)
INSERT INTO game_public_lobby_question (position, question_id)
SELECT position, question_id
FROM resolved
ON CONFLICT (position) DO UPDATE
SET question_id = EXCLUDED.question_id,
    updated_at = NOW();

WITH seed(position, question_text_ru, question_text_en, correct_answer) AS (
    VALUES
        (1, 'Сколько планет в Солнечной системе?', 'How many planets are there in the Solar System?', 8::DOUBLE PRECISION),
        (2, 'В каком году человек впервые высадился на Луне?', 'In what year did humans first land on the Moon?', 1969::DOUBLE PRECISION),
        (3, 'Сколько колец на олимпийском флаге?', 'How many rings are on the Olympic flag?', 5::DOUBLE PRECISION),
        (4, 'Сколько клеток на шахматной доске?', 'How many squares are there on a chessboard?', 64::DOUBLE PRECISION),
        (5, 'В каком году Apple представила первый iPhone?', 'In what year did Apple introduce the first iPhone?', 2007::DOUBLE PRECISION),
        (6, 'Сколько химических элементов официально входит в периодическую таблицу?', 'How many chemical elements are officially in the periodic table?', 118::DOUBLE PRECISION),
        (7, 'В каком году затонул Титаник?', 'In what year did the Titanic sink?', 1912::DOUBLE PRECISION),
        (8, 'Сколько фильмов входит в основную серию о Гарри Поттере?', 'How many films are in the main Harry Potter series?', 8::DOUBLE PRECISION)
)
UPDATE game_question q
SET question_text = s.question_text_ru,
    question_text_ru = s.question_text_ru,
    question_text_en = s.question_text_en,
    correct_answer = s.correct_answer,
    is_active = TRUE,
    updated_at = NOW()
FROM seed s
JOIN game_public_lobby_question plq ON plq.position = s.position
WHERE q.id = plq.question_id;
