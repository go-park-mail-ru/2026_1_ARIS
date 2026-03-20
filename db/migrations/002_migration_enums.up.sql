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

CREATE TYPE reaction_type AS ENUM (
    '\like',
    '\dislike',
    '\anger',
    '\happy'
)