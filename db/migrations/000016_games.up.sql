CREATE TABLE IF NOT EXISTS game_question (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID NOT NULL,
    game_type TEXT NOT NULL DEFAULT 'number_duel' CHECK (LENGTH(game_type) >= 1 AND LENGTH(game_type) <= 64),
    slug TEXT NOT NULL UNIQUE CHECK (LENGTH(slug) >= 3 AND LENGTH(slug) <= 96),
    question_text TEXT NOT NULL CHECK (LENGTH(question_text) >= 5 AND LENGTH(question_text) <= 512),
    correct_answer DOUBLE PRECISION NOT NULL,
    answer_unit TEXT DEFAULT NULL CHECK (answer_unit IS NULL OR LENGTH(answer_unit) <= 64),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS game_room (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID NOT NULL,
    invite_code TEXT NOT NULL UNIQUE CHECK (LENGTH(invite_code) >= 4 AND LENGTH(invite_code) <= 16),
    game_type TEXT NOT NULL DEFAULT 'number_duel' CHECK (LENGTH(game_type) >= 1 AND LENGTH(game_type) <= 64),
    status TEXT NOT NULL DEFAULT 'waiting' CHECK (status IN ('waiting', 'active', 'finished')),
    created_by_profile_id BIGINT NOT NULL REFERENCES profile(id),
    winner_profile_id BIGINT REFERENCES profile(id),
    question_count INT NOT NULL DEFAULT 5 CHECK (question_count >= 1 AND question_count <= 25),
    answer_timeout_sec INT NOT NULL DEFAULT 10 CHECK (answer_timeout_sec >= 3 AND answer_timeout_sec <= 120),
    current_question_index INT NOT NULL DEFAULT 0 CHECK (current_question_index >= 0),
    current_question_id BIGINT REFERENCES game_question(id),
    question_started_at TIMESTAMPTZ,
    question_deadline_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS game_room_member (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID NOT NULL,
    room_id BIGINT NOT NULL REFERENCES game_room(id) ON DELETE CASCADE,
    profile_id BIGINT NOT NULL REFERENCES profile(id),
    score INT NOT NULL DEFAULT 0 CHECK (score >= 0),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT game_room_member_unique UNIQUE (room_id, profile_id)
);

CREATE TABLE IF NOT EXISTS game_room_question (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID NOT NULL,
    room_id BIGINT NOT NULL REFERENCES game_room(id) ON DELETE CASCADE,
    question_id BIGINT NOT NULL REFERENCES game_question(id),
    position INT NOT NULL CHECK (position >= 1),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'completed')),
    winner_profile_id BIGINT REFERENCES profile(id),
    started_at TIMESTAMPTZ,
    deadline_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT game_room_question_unique UNIQUE (room_id, position),
    CONSTRAINT game_room_question_question_unique UNIQUE (room_id, question_id)
);

CREATE TABLE IF NOT EXISTS game_answer (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID NOT NULL,
    room_question_id BIGINT NOT NULL REFERENCES game_room_question(id) ON DELETE CASCADE,
    profile_id BIGINT NOT NULL REFERENCES profile(id),
    answer DOUBLE PRECISION NOT NULL,
    distance DOUBLE PRECISION NOT NULL CHECK (distance >= 0),
    answered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT game_answer_unique UNIQUE (room_question_id, profile_id)
);

CREATE INDEX IF NOT EXISTS idx_game_room_member_profile ON game_room_member(profile_id);
CREATE INDEX IF NOT EXISTS idx_game_room_status ON game_room(status);
CREATE INDEX IF NOT EXISTS idx_game_room_question_room ON game_room_question(room_id, position);
CREATE INDEX IF NOT EXISTS idx_game_answer_room_question ON game_answer(room_question_id);

INSERT INTO game_question (uid, game_type, slug, question_text, correct_answer, answer_unit)
VALUES
    ('11111111-1111-4111-8111-111111111111', 'number_duel', 'moon-landing-year', 'В каком году человек впервые высадился на Луне?', 1969, 'год'),
    ('11111111-1111-4111-8111-111111111112', 'number_duel', 'harry-potter-films', 'Сколько фильмов входит в основную серию о Гарри Поттере?', 8, 'фильмов'),
    ('11111111-1111-4111-8111-111111111113', 'number_duel', 'minecraft-release-year', 'В каком году вышла полная версия Minecraft 1.0?', 2011, 'год'),
    ('11111111-1111-4111-8111-111111111114', 'number_duel', 'marvel-infinity-stones', 'Сколько Камней Бесконечности в киновселенной Marvel?', 6, 'камней'),
    ('11111111-1111-4111-8111-111111111115', 'number_duel', 'periodic-table-elements', 'Сколько химических элементов официально входит в периодическую таблицу?', 118, 'элементов'),
    ('11111111-1111-4111-8111-111111111116', 'number_duel', 'titanic-sinking-year', 'В каком году затонул Титаник?', 1912, 'год'),
    ('11111111-1111-4111-8111-111111111117', 'number_duel', 'star-wars-skywalker-films', 'Сколько эпизодов входит в сагу Скайуокеров?', 9, 'эпизодов'),
    ('11111111-1111-4111-8111-111111111118', 'number_duel', 'first-iphone-year', 'В каком году Apple представила первый iPhone?', 2007, 'год'),
    ('11111111-1111-4111-8111-111111111119', 'number_duel', 'olympic-rings', 'Сколько колец на олимпийском флаге?', 5, 'колец'),
    ('11111111-1111-4111-8111-111111111120', 'number_duel', 'hobbit-book-year', 'В каком году впервые опубликован роман Хоббит?', 1937, 'год'),
    ('11111111-1111-4111-8111-111111111121', 'number_duel', 'chess-board-squares', 'Сколько клеток на шахматной доске?', 64, 'клеток'),
    ('11111111-1111-4111-8111-111111111122', 'number_duel', 'pokemon-gen-one', 'Сколько покемонов было в первом поколении?', 151, 'покемон')
ON CONFLICT (slug) DO UPDATE
SET question_text = EXCLUDED.question_text,
    correct_answer = EXCLUDED.correct_answer,
    answer_unit = EXCLUDED.answer_unit,
    is_active = TRUE,
    updated_at = NOW();
