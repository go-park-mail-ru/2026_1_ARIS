WITH public_seed(position, question_text_ru, question_text_en, correct_answer) AS (
    VALUES
        (1, 'Сколько портов существует в протоколе TCP?', 'How many ports exist in the TCP protocol?', 65536::DOUBLE PRECISION),
        (2, 'Какое среднее расстояние (в километрах) от Земли до Луны?', 'What is the average distance in kilometers from Earth to the Moon?', 384400::DOUBLE PRECISION),
        (3, 'Сколько элементов содержит современная периодическая таблица Менделеева?', 'How many elements are in the modern periodic table?', 118::DOUBLE PRECISION),
        (4, 'В каком году был основан Оксфордский университет?', 'In what year was the University of Oxford founded?', 1096::DOUBLE PRECISION),
        (5, 'Сколько животных каждого вида взял Моисей в ковчег?', 'How many animals of each kind did Moses take onto the ark?', 0::DOUBLE PRECISION)
),
updated_existing AS (
    UPDATE game_question AS q
    SET question_text = s.question_text_ru,
        question_text_ru = s.question_text_ru,
        question_text_en = s.question_text_en,
        correct_answer = s.correct_answer,
        is_active = TRUE,
        updated_at = NOW()
    FROM public_seed AS s
    JOIN game_public_lobby_question AS plq ON plq.position = s.position
    WHERE q.id = plq.question_id
    RETURNING s.position, q.id AS question_id
),
inserted_missing AS (
    INSERT INTO game_question (uid, game_type, question_text, question_text_ru, question_text_en, correct_answer, is_active)
    SELECT gen_random_uuid(), 'number_duel', s.question_text_ru, s.question_text_ru, s.question_text_en, s.correct_answer, TRUE
    FROM public_seed AS s
    WHERE NOT EXISTS (
        SELECT 1
        FROM updated_existing AS u
        WHERE u.position = s.position
    )
    RETURNING id, question_text_ru
),
resolved AS (
    SELECT position, question_id
    FROM updated_existing
    UNION ALL
    SELECT s.position, i.id
    FROM public_seed AS s
    JOIN inserted_missing AS i ON i.question_text_ru = s.question_text_ru
)
INSERT INTO game_public_lobby_question (position, question_id)
SELECT position, question_id
FROM resolved
ON CONFLICT (position) DO UPDATE
SET question_id = EXCLUDED.question_id,
    updated_at = NOW();

DELETE FROM game_public_lobby_question
WHERE position > 5;

UPDATE game_room
SET question_count = 5,
    updated_at = NOW()
WHERE is_public_lobby = TRUE
  AND status = 'waiting';
