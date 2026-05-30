ALTER TABLE game_room
    DROP CONSTRAINT IF EXISTS game_room_round_pause_sec_check;

ALTER TABLE game_room
    DROP COLUMN IF EXISTS round_pause_sec;
