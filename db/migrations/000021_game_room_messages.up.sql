CREATE TABLE IF NOT EXISTS game_room_message (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID NOT NULL,
    room_id BIGINT NOT NULL REFERENCES game_room(id) ON DELETE CASCADE,
    profile_id BIGINT NOT NULL REFERENCES profile(id),
    message_text TEXT NOT NULL CHECK (LENGTH(TRIM(message_text)) BETWEEN 1 AND 500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_game_room_message_room_created
    ON game_room_message(room_id, created_at, id);
