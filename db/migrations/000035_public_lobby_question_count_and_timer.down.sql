UPDATE game_room
SET question_count = 8,
    answer_timeout_sec = 15,
    updated_at = NOW()
WHERE is_public_lobby = TRUE
  AND status = 'waiting';
