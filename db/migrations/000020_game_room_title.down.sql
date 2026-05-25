DROP INDEX IF EXISTS game_room_active_waiting_title_unique_idx;

ALTER TABLE game_room
    DROP CONSTRAINT IF EXISTS game_room_title_length_check;

ALTER TABLE game_room
    DROP COLUMN IF EXISTS title;
