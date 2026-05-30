-- Keep public lobby question edits on rollback; only disable the extra
-- ordinary-room seed rows added by this migration so old room history remains valid.
UPDATE game_question
SET is_active = FALSE,
    updated_at = NOW()
WHERE uid::TEXT LIKE '33333333-3333-4333-8333-%';
