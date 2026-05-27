ALTER TABLE game_room
    ADD COLUMN IF NOT EXISTS is_ranked BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS game_rating_season (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID NOT NULL,
    game_type TEXT NOT NULL DEFAULT 'number_duel' CHECK (LENGTH(game_type) >= 1 AND LENGTH(game_type) <= 64),
    season_number INT NOT NULL CHECK (season_number >= 1),
    season_year INT NOT NULL CHECK (season_year >= 2026),
    season_month INT NOT NULL CHECK (season_month >= 1 AND season_month <= 12),
    title TEXT NOT NULL CHECK (LENGTH(title) >= 3 AND LENGTH(title) <= 64),
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT game_rating_season_period_check CHECK (ends_at > starts_at),
    CONSTRAINT game_rating_season_month_unique UNIQUE (game_type, season_year, season_month),
    CONSTRAINT game_rating_season_number_unique UNIQUE (game_type, season_number)
);

CREATE TABLE IF NOT EXISTS game_player_rating (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID NOT NULL,
    season_id BIGINT NOT NULL REFERENCES game_rating_season(id) ON DELETE CASCADE,
    game_type TEXT NOT NULL DEFAULT 'number_duel' CHECK (LENGTH(game_type) >= 1 AND LENGTH(game_type) <= 64),
    profile_id BIGINT NOT NULL REFERENCES profile(id),
    rating INT NOT NULL DEFAULT 1000 CHECK (rating >= 0),
    games_played INT NOT NULL DEFAULT 0 CHECK (games_played >= 0),
    wins INT NOT NULL DEFAULT 0 CHECK (wins >= 0),
    draws INT NOT NULL DEFAULT 0 CHECK (draws >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT game_player_rating_unique UNIQUE (season_id, profile_id)
);

CREATE TABLE IF NOT EXISTS game_rating_match (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID NOT NULL,
    room_id BIGINT NOT NULL REFERENCES game_room(id) ON DELETE CASCADE,
    season_id BIGINT NOT NULL REFERENCES game_rating_season(id) ON DELETE CASCADE,
    game_type TEXT NOT NULL DEFAULT 'number_duel' CHECK (LENGTH(game_type) >= 1 AND LENGTH(game_type) <= 64),
    group_hash TEXT NOT NULL CHECK (LENGTH(group_hash) >= 16 AND LENGTH(group_hash) <= 128),
    group_occurrence INT NOT NULL CHECK (group_occurrence >= 1),
    rating_weight DOUBLE PRECISION NOT NULL CHECK (rating_weight >= 0 AND rating_weight <= 1),
    played_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT game_rating_match_room_unique UNIQUE (room_id)
);

CREATE TABLE IF NOT EXISTS game_rating_match_player (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    match_id BIGINT NOT NULL REFERENCES game_rating_match(id) ON DELETE CASCADE,
    profile_id BIGINT NOT NULL REFERENCES profile(id),
    score INT NOT NULL DEFAULT 0 CHECK (score >= 0),
    place INT NOT NULL CHECK (place >= 1),
    before_rating INT NOT NULL CHECK (before_rating >= 0),
    after_rating INT NOT NULL CHECK (after_rating >= 0),
    rating_delta INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT game_rating_match_player_unique UNIQUE (match_id, profile_id)
);

CREATE INDEX IF NOT EXISTS idx_game_player_rating_leaderboard
    ON game_player_rating(season_id, rating DESC, games_played DESC);

CREATE INDEX IF NOT EXISTS idx_game_rating_match_group_day
    ON game_rating_match(game_type, group_hash, played_at);
