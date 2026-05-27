ALTER TABLE game_question
    DROP CONSTRAINT IF EXISTS game_question_slug_check,
    DROP CONSTRAINT IF EXISTS game_question_slug_key;

ALTER TABLE game_question
    DROP COLUMN IF EXISTS slug;
