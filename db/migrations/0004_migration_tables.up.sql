CREATE TABLE IF NOT EXISTS user_account (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID UNIQUE NOT NULL,
    email TEXT UNIQUE CHECK (
        email ~ '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$' AND LENGTH(email) >= 5 AND LENGTH(email) <= 255
    ),
    phone TEXT UNIQUE CHECK (phone ~ '^\+?\d{4,15}$' AND LENGTH(phone) >= 11 AND LENGTH(phone) <= 16),
    password_hash TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT user_account_contact_check CHECK (
        email IS NOT NULL
        OR phone IS NOT NULL
    )
);

CREATE TABLE IF NOT EXISTS media (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID UNIQUE NOT NULL,
    media_name TEXT NOT NULL CHECK(LENGTH(media_name) <= 255 AND LENGTH(media_name) >= 1),
    extension TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    description TEXT DEFAULT NULL CHECK(LENGTH(description) <= 255),
    size BIGINT NOT NULL CHECK (size >= 0 AND size <= 1024*1024*1024), -- гигабайт
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT default_description CHECK (
        description IS NULL
        OR description <> ''
    )
);

CREATE TABLE IF NOT EXISTS profile (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID UNIQUE NOT NULL,
    avatar_id BIGINT REFERENCES media(id) ON DELETE SET NULL,
    username TEXT NOT NULL UNIQUE CHECK(LENGTH(username) >= 3 AND LENGTH(username) <= 20),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_profile (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID UNIQUE NOT NULL,
    user_account_id BIGINT unique NOT NULL REFERENCES user_account(id) ON DELETE CASCADE,
    profile_id BIGINT unique NOT NULL REFERENCES profile(id) ON DELETE CASCADE,
    first_name TEXT NOT NULL CHECK(LENGTH(first_name) >= 1 AND LENGTH(first_name) <= 255 AND first_name ~ '^[A-Za-zА-Яа-яЁё]{1,}$'),
    last_name TEXT NOT NULL CHECK(LENGTH(last_name) >= 1 AND LENGTH(last_name) <= 255 AND last_name ~ '^[A-Za-zА-Яа-яЁё]{1,}$'),
    bio TEXT DEFAULT NULL CHECK(LENGTH(bio) <= 1023),
    birthday_date DATE,
    gender gender_type NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT default_bio CHECK (
        bio IS NULL
        OR bio <> ''
    )
);

CREATE TABLE IF NOT EXISTS post (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID UNIQUE NOT NULL,
    post_text TEXT DEFAULT NULL CHECK(LENGTH(post_text) <= 5000),
    author_id BIGINT NOT NULL REFERENCES profile(id),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT default_text CHECK (
        post_text IS NULL
        OR post_text <> ''
    )
);

CREATE TABLE IF NOT EXISTS post_with_media (
    post_id BIGINT NOT NULL REFERENCES post(id) ON DELETE CASCADE,
    media_id BIGINT NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    sort_order INT NOT NULL DEFAULT 0 CHECK (sort_order >= 0 AND sort_order <= 10),
    CONSTRAINT unique_order_per_post UNIQUE (post_id, sort_order),
    PRIMARY KEY (post_id, media_id)
);

CREATE TABLE IF NOT EXISTS chat (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID UNIQUE NOT NULL,
    chat_type chat_type NOT NULL,
    title TEXT NOT NULL CHECK(LENGTH(title) >= 1 AND LENGTH(title) <= 63),
    avatar_id BIGINT REFERENCES media(id) ON DELETE SET NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT default_title CHECK (title <> '')
);

CREATE TABLE IF NOT EXISTS chat_member (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID UNIQUE NOT NULL,
    chat_id BIGINT NOT NULL REFERENCES chat(id) ON DELETE CASCADE,
    profile_id BIGINT NOT NULL REFERENCES profile(id),
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
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID UNIQUE NOT NULL,
    title TEXT NOT NULL CHECK(LENGTH(title) >= 1 AND LENGTH(title) <= 63),
    author_id BIGINT REFERENCES profile(id),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT default_title CHECK (title <> '')
);

CREATE TABLE IF NOT EXISTS sticker (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID UNIQUE NOT NULL,
    size BIGINT NOT NULL CHECK (size >= 0 AND size <= 1024*1024),
    sort_order INT NOT NULL DEFAULT 0 CHECK (sort_order >= 0 AND sort_order <= 100),
    pack_id BIGINT REFERENCES sticker_pack(id) ON DELETE SET NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_index_per_pack UNIQUE (pack_id, sort_order)
);

CREATE TABLE IF NOT EXISTS message (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID UNIQUE NOT NULL,
    message_text TEXT CHEcK(LENGTH(message_text) >= 1 AND LENGTH(message_text) <= 5000),
    parent_message_id BIGINT REFERENCES message(id) ON DELETE SET NULL,
    chat_id BIGINT NOT NULL REFERENCES chat(id) ON DELETE CASCADE,
    status message_status NOT NULL DEFAULT 'pending',
    sticker_id BIGINT REFERENCES sticker(id) ON DELETE SET NULL,
    author_id BIGINT NOT NULL REFERENCES profile(id),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT message_content_check CHECK (
        message_text IS NOT NULL AND message_text <> '' AND sticker_id IS NULL
        OR 
        sticker_id IS NOT NULL AND message_text IS NULL
    )
);

CREATE TABLE IF NOT EXISTS message_with_media (
    message_id BIGINT NOT NULL REFERENCES message(id) ON DELETE CASCADE,
    media_id BIGINT NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    sort_order INT NOT NULL DEFAULT 0 CHECK (sort_order >= 0 AND sort_order <= 10),
    PRIMARY KEY (message_id, media_id),
    CONSTRAINT unique_order_per_message UNIQUE (message_id, sort_order)
);

CREATE TABLE IF NOT EXISTS community (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID UNIQUE NOT NULL,
    title TEXT NOT NULL CHECK(LENGTH(title) >= 1 AND LENGTH(title) <= 64),
    bio TEXT CHECK(LENGTH(bio) <= 2047),
    community_type community_type NOT NULL DEFAULT 'public',
    profile_id BIGINT NOT NULL UNIQUE REFERENCES profile(id) ON DELETE CASCADE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT default_title CHECK (title <> ''),
    CONSTRAINT default_bio CHECK (
        bio IS NULL
        OR bio <> ''
    )
);

CREATE TABLE IF NOT EXISTS community_member (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID UNIQUE NOT NULL,
    profile_id BIGINT NOT NULL REFERENCES profile(id),
    community_id BIGINT NOT NULL REFERENCES community(id) ON DELETE CASCADE,
    community_role community_member_role NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    leave_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT community_member_unique UNIQUE (profile_id, community_id),
    CONSTRAINT community_member_leave_after_join CHECK (
        leave_at IS NULL
        OR leave_at > joined_at
    )
);

CREATE TABLE IF NOT EXISTS comment (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID UNIQUE NOT NULL,
    comment_text TEXT,
    post_id BIGINT NOT NULL REFERENCES post(id) ON DELETE CASCADE,
    parent_comment_id BIGINT REFERENCES comment(id) ON DELETE SET NULL,
    sticker_id BIGINT REFERENCES sticker(id) ON DELETE SET NULL,
    author_id BIGINT NOT NULL REFERENCES profile(id),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT comment_content_check CHECK (
        comment_text IS NOT NULL AND comment_text <> '' AND sticker_id is NULL
        OR
        sticker_id IS NOT NULL AND comment_text IS NULL
    )
);

CREATE TABLE IF NOT EXISTS comment_with_media (
    comment_id BIGINT NOT NULL REFERENCES comment(id) ON DELETE CASCADE,
    media_id BIGINT NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    sort_order INT NOT NULL DEFAULT 0 CHECK (sort_order >= 0 AND sort_order <= 10),
    CONSTRAINT unique_order_per_comment UNIQUE (comment_id, sort_order),
    PRIMARY KEY (comment_id, media_id)
);

CREATE TABLE IF NOT EXISTS like_record (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID UNIQUE NOT NULL,
    post_id BIGINT REFERENCES post(id),
    comment_id BIGINT REFERENCES comment(id),
    author_id BIGINT NOT NULL REFERENCES profile(id),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT like_targer_check CHECK (
        (post_id IS NOT NULL) <> (comment_id IS NOT NULL)
    )
);

CREATE TABLE IF NOT EXISTS reaction (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID UNIQUE NOT NULL,
    message_id BIGINT NOT NULL REFERENCES message(id) ON DELETE CASCADE,
    reaction_type reaction_type NOT NULL,
    author_id BIGINT NOT NULL REFERENCES profile(id),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT reaction_unique UNIQUE (message_id, author_id)
);

CREATE TABLE IF NOT EXISTS friendship (
    friend1_id BIGINT NOT NULL REFERENCES profile(id),
    friend2_id BIGINT NOT NULL REFERENCES profile(id),
    requester_id BIGINT NOT NULL REFERENCES profile(id),
    status friendship_status NOT NULL DEFAULT 'pending',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT friendship_no_self_reference CHECK (friend1_id <> friend2_id),
    CONSTRAINT friendship_order CHECK (friend1_id < friend2_id),
    CONSTRAINT requester CHECK(requester_id = friend1_id OR requester_id = friend2_id)
    PRIMARY KEY (friend1_id, friend2_id)
);

CREATE TABLE IF NOT EXISTS ad (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID UNIQUE NOT NULL,
    title TEXT NOT NULL CHECK(LENGTH(title) >= 1 AND LENGTH(title) <= 1023),
    description TEXT CHECK(LENGTH(description) <= 2047),
    link TEXT NOT NULL,
    media_id BIGINT REFERENCES media (id) ON DELETE SET NULL,
    author_id BIGINT NOT NULL REFERENCES profile(id) ON DELETE RESTRICT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ad_meta (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid UUID UNIQUE NOT NULL,
    ad_id BIGINT NOT NULL REFERENCES ad(id) ON DELETE CASCADE,
    meta_key TEXT NOT NULL CHECK(LENGTH(meta_key) >= 1 AND LENGTH(meta_key) <= 255),
    meta_value TEXT NOT NULL CHECK(LENGTH(meta_value) >=1 AND LENGTH(meta_value) <= 255),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);