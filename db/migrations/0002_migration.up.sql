CREATE TYPE gender_type AS ENUM ('male', 'female');

CREATE TYPE chat_type AS ENUM ('personal', 'community');

CREATE TYPE chat_member_role AS ENUM ('admin', 'member');

CREATE TYPE community_type AS ENUM ('public', 'private');

CREATE TYPE community_member_role AS ENUM (
    'owner',
    'admin',
    'moderator',
    'member'
);

CREATE TYPE friendship_status AS ENUM ('pending', 'accepted', 'declined');

CREATE TYPE message_status AS ENUM (
    'pending',
    'sent',
    'delivered',
    'read'
);

CREATE
OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
NEW.updated_at = NOW();
RETURN NEW;
END;
$$;

CREATE TABLE IF NOT EXISTS user_account (
    id UUID PRIMARY KEY,
    email TEXT UNIQUE CHECK (
        email ~ '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$'
    ),
    phone TEXT UNIQUE CHECK (phone ~ '^\+7[0-9]{10}$'),
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT user_account_contact_check CHECK (
        email IS NOT NULL
        OR phone IS NOT NULL
    )
);

CREATE TABLE IF NOT EXISTS media (
    id UUID PRIMARY KEY,
    media_name TEXT NOT NULL,
    extension TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    description TEXT DEFAULT NULL,
    object_key TEXT NOT NULL UNIQUE,
    size BIGINT NOT NULL CHECK (size >= 0),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT default_description CHECK (
        description IS NULL
        OR description <> ''
    )
);

CREATE TABLE IF NOT EXISTS profile (
    id UUID PRIMARY KEY,
    avatar_id UUID REFERENCES media(id) ON DELETE
    SET
        NULL,
        username TEXT NOT NULL UNIQUE,
        is_active BOOLEAN NOT NULL DEFAULT TRUE,
        created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_profile (
    id UUID primary key,
    user_account_id UUID NOT NULL REFERENCES user_account(id) ON DELETE CASCADE,
    profile_id UUID NOT NULL REFERENCES profile(id) ON DELETE CASCADE,
    first_name TEXT,
    last_name TEXT,
    bio TEXT DEFAULT NULL,
    birthday_date DATE,
    gender gender_type NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT default_bio CHECK (
        bio IS NULL
        OR bio <> ''
    )
);

CREATE TABLE IF NOT EXISTS post (
    id UUID PRIMARY KEY,
    post_text TEXT DEFAULT NULL,
    author_id UUID NOT NULL REFERENCES profile(id),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT default_text CHECK (
        post_text IS NULL
        OR post_text <> ''
    )
);

CREATE TABLE IF NOT EXISTS post_with_media (
    post_id UUID NOT NULL REFERENCES post(id) ON DELETE CASCADE,
    media_id UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    sort_order INT NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    PRIMARY KEY (post_id, media_id),
    CONSTRAINT unique_order_per_post UNIQUE (post_id, sort_order)
);

CREATE TABLE IF NOT EXISTS chat (
    id UUID PRIMARY KEY,
    chat_type chat_type NOT NULL,
    title TEXT NOT NULL,
    avatar_id UUID REFERENCES media(id) ON DELETE
    SET
        NULL,
        is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        CONSTRAINT default_title CHECK (title <> '')
);

CREATE TABLE IF NOT EXISTS chat_member (
    id UUID PRIMARY KEY,
    chat_id UUID NOT NULL REFERENCES chat(id) ON DELETE CASCADE,
    profile_id UUID NOT NULL REFERENCES profile(id),
    chat_role chat_member_role NOT NULL DEFAULT 'member',
    joined_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    leave_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT chat_member_unique UNIQUE (chat_id, profile_id),
    CONSTRAINT chat_member_leave_after_join CHECK (
        leave_at IS NULL
        OR leave_at > joined_at
    )
);

CREATE TABLE IF NOT EXISTS sticker_pack (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    author_id UUID REFERENCES profile(id),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT default_title CHECK (title <> '')
);

CREATE TABLE IF NOT EXISTS sticker (
    id UUID PRIMARY KEY,
    link TEXT NOT NULL UNIQUE,
    size BIGINT NOT NULL CHECK (size >= 0),
    index_order INT NOT NULL DEFAULT 0 CHECK (index_order >= 0),
    pack_id UUID NOT NULL REFERENCES sticker_pack(id) ON DELETE CASCADE,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_index_per_pack UNIQUE (pack_id, index_order)
);

CREATE TABLE IF NOT EXISTS message (
    id UUID PRIMARY KEY,
    message_text TEXT,
    parent_message_id UUID REFERENCES message(id) ON DELETE
    SET
        NULL,
        chat_id UUID NOT NULL REFERENCES chat(id) ON DELETE CASCADE,
        status message_status NOT NULL DEFAULT 'sent',
        sticker_id UUID REFERENCES sticker(id) ON DELETE
    SET
        NULL,
        author_id UUID NOT NULL REFERENCES profile(id),
        is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        CONSTRAINT message_content_check CHECK (
            message_text IS NOT NULL
            AND message_text <> ''
            OR sticker_id IS NOT NULL
        )
);

CREATE TABLE IF NOT EXISTS message_with_media (
    message_id UUID NOT NULL REFERENCES message(id) ON DELETE CASCADE,
    media_id UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    sort_order INT NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    PRIMARY KEY (message_id, media_id),
    CONSTRAINT unique_order_per_message UNIQUE (message_id, sort_order)
);

CREATE TABLE IF NOT EXISTS community (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    bio TEXT,
    community_type community_type NOT NULL DEFAULT 'public',
    owner_id UUID NOT NULL REFERENCES profile(id),
    profile_id UUID NOT NULL REFERENCES profile(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT default_title CHECK (title <> ''),
    CONSTRAINT default_bio CHECK (
        bio IS NULL
        OR bio <> ''
    )
);

CREATE TABLE IF NOT EXISTS community_member (
    id UUID PRIMARY KEY,
    profile_id UUID NOT NULL REFERENCES profile(id),
    community_id UUID NOT NULL REFERENCES community(id) ON DELETE CASCADE,
    community_role community_member_role NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    leave_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT community_member_unique UNIQUE (profile_id, community_id),
    CONSTRAINT community_member_leave_after_join CHECK (
        leave_at IS NULL
        OR leave_at > joined_at
    )
);

CREATE TABLE IF NOT EXISTS comment (
    id UUID PRIMARY KEY,
    comment_text TEXT,
    post_id UUID NOT NULL REFERENCES post(id) ON DELETE CASCADE,
    parent_comment_id UUID REFERENCES comment(id) ON DELETE
    SET
        NULL,
        sticker_id UUID REFERENCES sticker(id) ON DELETE
    SET
        NULL,
        author_id UUID NOT NULL REFERENCES profile(id),
        is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        CONSTRAINT comment_content_check CHECK (
            comment_text IS NOT NULL
            AND comment_text <> ''
            OR sticker_id IS NOT NULL
        )
);

CREATE TABLE IF NOT EXISTS comment_with_media (
    comment_id UUID NOT NULL REFERENCES comment(id) ON DELETE CASCADE,
    media_id UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    sort_order INT NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    PRIMARY KEY (comment_id, media_id),
    CONSTRAINT unique_order_per_comment UNIQUE (comment_id, sort_order)
);

CREATE TABLE IF NOT EXISTS like_record (
    id UUID PRIMARY KEY,
    author_id UUID NOT NULL REFERENCES profile(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS like_to_post (
    like_id UUID NOT NULL UNIQUE REFERENCES like_record(id) ON DELETE CASCADE,
    post_id UUID NOT NULL REFERENCES post(id) ON DELETE CASCADE,
    PRIMARY KEY (like_id, post_id)
);

CREATE TABLE IF NOT EXISTS like_to_comment (
    like_id UUID NOT NULL UNIQUE REFERENCES like_record(id) ON DELETE CASCADE,
    comment_id UUID NOT NULL REFERENCES comment(id) ON DELETE CASCADE,
    PRIMARY KEY (like_id, comment_id)
);

CREATE TABLE IF NOT EXISTS reaction (
    id UUID PRIMARY KEY,
    message_id UUID NOT NULL REFERENCES message(id) ON DELETE CASCADE,
    reaction_type TEXT NOT NULL,
    author_id UUID NOT NULL REFERENCES profile(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT reaction_unique UNIQUE (message_id, author_id)
);

CREATE TABLE IF NOT EXISTS friendship (
    friend1_id UUID NOT NULL REFERENCES profile(id),
    friend2_id UUID NOT NULL REFERENCES profile(id),
    status friendship_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (friend1_id, friend2_id),
    CONSTRAINT friendship_no_self_reference CHECK (friend1_id <> friend2_id),
    CONSTRAINT friendship_order CHECK (friend1_id < friend2_id)
);

CREATE TABLE IF NOT EXISTS ad (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    link TEXT NOT NULL,
    media_id UUID REFERENCES media (id) ON DELETE
    SET
        NULL,
        author_id UUID NOT NULL REFERENCES profile(id),
        is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
        created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ad_meta (
    id UUID PRIMARY KEY,
    ad_id UUID NOT NULL REFERENCES ad(id) ON DELETE CASCADE,
    meta_key TEXT NOT NULL,
    meta_value TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

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