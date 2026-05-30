ALTER TABLE game_room
    ADD COLUMN IF NOT EXISTS round_pause_sec INT;

UPDATE game_room
SET round_pause_sec = 5
WHERE round_pause_sec IS NULL;

ALTER TABLE game_room
    ALTER COLUMN round_pause_sec SET DEFAULT 5,
    ALTER COLUMN round_pause_sec SET NOT NULL;

ALTER TABLE game_room
    DROP CONSTRAINT IF EXISTS game_room_round_pause_sec_check;

ALTER TABLE game_room
    ADD CONSTRAINT game_room_round_pause_sec_check
    CHECK (round_pause_sec >= 1 AND round_pause_sec <= 60);
