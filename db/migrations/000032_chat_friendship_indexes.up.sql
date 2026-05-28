CREATE INDEX IF NOT EXISTS message_active_chat_created_idx
    ON message (chat_id, created_at DESC, id DESC)
    WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS message_active_chat_id_idx
    ON message (chat_id, id ASC)
    WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS friendship_requester_status_idx
    ON friendship (requester_id, status, addressee_id);

CREATE INDEX IF NOT EXISTS friendship_addressee_status_idx
    ON friendship (addressee_id, status, requester_id);
