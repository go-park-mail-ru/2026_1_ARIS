CREATE TRIGGER trg_user_account_updated_at BEFORE
UPDATE
    ON user_account FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_media_updated_at BEFORE
UPDATE
    ON media FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_profile_updated_at BEFORE
UPDATE
    ON profile FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_user_profile_updated_at BEFORE
UPDATE
    ON user_profile FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_post_updated_at BEFORE
UPDATE
    ON post FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_chat_updated_at BEFORE
UPDATE
    ON chat FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_chat_member_updated_at BEFORE
UPDATE
    ON chat_member FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_sticker_pack_updated_at BEFORE
UPDATE
    ON sticker_pack FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_sticker_updated_at BEFORE
UPDATE
    ON sticker FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_message_updated_at BEFORE
UPDATE
    ON message FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_community_updated_at BEFORE
UPDATE
    ON community FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_community_member_updated_at BEFORE
UPDATE
    ON community_member FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_comment_updated_at BEFORE
UPDATE
    ON comment FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_reaction_updated_at BEFORE
UPDATE
    ON like_record FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_reaction_updated_at BEFORE
UPDATE
    ON reaction FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_friendship_updated_at BEFORE
UPDATE
    ON friendship FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_ad_updated_at BEFORE
UPDATE
    ON ad FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_ad_meta_updated_at BEFORE
UPDATE
    ON ad_meta FOR EACH ROW EXECUTE FUNCTION set_updated_at();