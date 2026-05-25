ALTER TABLE game_room_member
    DROP COLUMN IF EXISTS force_resume_requested,
    DROP COLUMN IF EXISTS pause_used;

ALTER TABLE game_room
    DROP COLUMN IF EXISTS pause_until_at,
    DROP COLUMN IF EXISTS pause_started_at,
    DROP COLUMN IF EXISTS paused_by_profile_id,
    DROP COLUMN IF EXISTS next_question_at;
