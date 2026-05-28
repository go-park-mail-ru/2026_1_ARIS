SELECT id, uid, message_text, parent_message_id, chat_id, author_id, sticker_id,
       message_type, is_active, created_at, updated_at
FROM (
  SELECT id, uid, message_text, parent_message_id, chat_id, author_id, sticker_id,
         message_type, is_active, created_at, updated_at
  FROM message
  WHERE chat_id = (
    SELECT id FROM chat WHERE title = 'perf-chat-history' ORDER BY id DESC LIMIT 1
  )
    AND is_active = TRUE
  ORDER BY created_at DESC
  LIMIT 50 OFFSET 0
) latest_messages
ORDER BY created_at ASC;
