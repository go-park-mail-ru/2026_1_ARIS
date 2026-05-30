UPDATE game_room
SET question_count = 5,
    answer_timeout_sec = 7,
    updated_at = NOW()
WHERE is_public_lobby = TRUE
  AND status = 'waiting';
