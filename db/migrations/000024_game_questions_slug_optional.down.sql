ALTER TABLE game_question
    ADD COLUMN slug TEXT;

UPDATE game_question
SET slug = 'question-' || id
WHERE slug IS NULL OR LENGTH(slug) < 3;

ALTER TABLE game_question
    ALTER COLUMN slug SET NOT NULL;

ALTER TABLE game_question
    ADD CONSTRAINT game_question_slug_check CHECK (LENGTH(slug) >= 3 AND LENGTH(slug) <= 96);

ALTER TABLE game_question
    ADD CONSTRAINT game_question_slug_key UNIQUE (slug);
