ALTER TABLE message
  ADD COLUMN message_type TEXT NOT NULL DEFAULT 'text';

ALTER TABLE message
  ADD CONSTRAINT message_type_check
  CHECK (message_type IN ('text', 'video_note', 'sticker'));

UPDATE message
  SET message_type = 'sticker'
  WHERE sticker_id IS NOT NULL;
