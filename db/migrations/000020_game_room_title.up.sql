ALTER TABLE game_room
    ADD COLUMN IF NOT EXISTS title TEXT;

UPDATE game_room
SET title = invite_code
WHERE title IS NULL OR BTRIM(title) = '';

ALTER TABLE game_room
    ALTER COLUMN title SET NOT NULL;

ALTER TABLE game_room
    ADD CONSTRAINT game_room_title_length_check
        CHECK (LENGTH(BTRIM(title)) >= 1 AND LENGTH(title) <= 30);

CREATE UNIQUE INDEX IF NOT EXISTS game_room_active_waiting_title_unique_idx
    ON game_room (LOWER(title))
    WHERE is_active = TRUE AND status = 'waiting';
