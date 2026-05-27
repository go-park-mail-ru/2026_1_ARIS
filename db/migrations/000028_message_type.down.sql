ALTER TABLE message DROP CONSTRAINT IF EXISTS message_type_check;
ALTER TABLE message DROP COLUMN IF EXISTS message_type;
