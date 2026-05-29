ALTER TABLE game_room
    ADD COLUMN IF NOT EXISTS max_players INT;

UPDATE game_room
SET max_players = 2
WHERE max_players IS NULL;

ALTER TABLE game_room
    ALTER COLUMN max_players SET DEFAULT 2,
    ALTER COLUMN max_players SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'game_room_max_players_check'
          AND conrelid = 'game_room'::regclass
    ) THEN
        ALTER TABLE game_room
            ADD CONSTRAINT game_room_max_players_check
            CHECK (max_players >= 2 AND max_players <= 8);
    END IF;
END $$;

ALTER TABLE game_room
    ADD COLUMN IF NOT EXISTS password_hash TEXT;

ALTER TABLE game_room_member
    ADD COLUMN IF NOT EXISTS is_ready BOOLEAN,
    ADD COLUMN IF NOT EXISTS pause_used BOOLEAN,
    ADD COLUMN IF NOT EXISTS force_resume_requested BOOLEAN;

UPDATE game_room_member
SET is_ready = FALSE
WHERE is_ready IS NULL;

UPDATE game_room_member
SET pause_used = FALSE
WHERE pause_used IS NULL;

UPDATE game_room_member
SET force_resume_requested = FALSE
WHERE force_resume_requested IS NULL;

ALTER TABLE game_room_member
    ALTER COLUMN is_ready SET DEFAULT FALSE,
    ALTER COLUMN is_ready SET NOT NULL,
    ALTER COLUMN pause_used SET DEFAULT FALSE,
    ALTER COLUMN pause_used SET NOT NULL,
    ALTER COLUMN force_resume_requested SET DEFAULT FALSE,
    ALTER COLUMN force_resume_requested SET NOT NULL;
