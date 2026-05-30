UPDATE game_room
SET answer_timeout_sec = 10,
    round_pause_sec = 14,
    updated_at = NOW()
WHERE is_public_lobby = TRUE
  AND status = 'waiting';
