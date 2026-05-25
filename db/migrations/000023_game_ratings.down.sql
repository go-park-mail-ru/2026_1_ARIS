DROP TABLE IF EXISTS game_rating_match_player;
DROP TABLE IF EXISTS game_rating_match;
DROP TABLE IF EXISTS game_player_rating;
DROP TABLE IF EXISTS game_rating_season;

ALTER TABLE game_room
    DROP COLUMN IF EXISTS is_ranked;
