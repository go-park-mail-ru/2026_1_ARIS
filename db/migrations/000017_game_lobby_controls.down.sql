ALTER TABLE game_room_member
    DROP COLUMN IF EXISTS is_ready;

ALTER TABLE game_room
    DROP COLUMN IF EXISTS password_hash;
