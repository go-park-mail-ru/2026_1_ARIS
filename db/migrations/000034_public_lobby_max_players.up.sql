ALTER TABLE game_room
    DROP CONSTRAINT IF EXISTS game_room_max_players_check;

ALTER TABLE game_room
    ADD CONSTRAINT game_room_max_players_check
    CHECK (max_players >= 2 AND max_players <= 80);
