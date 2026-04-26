DROP TABLE IF EXISTS support_ticket_message CASCADE;
DROP TABLE IF EXISTS ticket_with_media CASCADE;

DROP INDEX IF EXISTS idx_support_ticket_rating;
DROP INDEX IF EXISTS idx_support_ticket_assigned_agent_id;
DROP INDEX IF EXISTS idx_support_ticket_line;

ALTER TABLE support_ticket
    DROP COLUMN IF EXISTS rating,
    DROP COLUMN IF EXISTS assigned_agent_id,
    DROP COLUMN IF EXISTS line,
    DROP COLUMN IF EXISTS uid;

ALTER TABLE support_profile_role
    DROP CONSTRAINT IF EXISTS support_profile_role_role_check;

UPDATE support_profile_role
SET role = 'support'
WHERE role IN ('support_l1', 'support_l2');

ALTER TABLE support_profile_role
    ADD CONSTRAINT support_profile_role_role_check
    CHECK (role IN ('admin', 'support'));
