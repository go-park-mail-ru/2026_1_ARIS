-- ClickHouse DDL for the recommendation system.
-- Safe to run multiple times (all statements are IF NOT EXISTS).
-- Executed automatically by the ClickHouse Docker image on first start
-- (initdb.d) and can also be applied manually to an existing instance.

CREATE DATABASE IF NOT EXISTS aris;

USE aris;

CREATE TABLE IF NOT EXISTS post_events
(
    event_date          Date,
    event_time          DateTime64(3),
    profile_id          Int64,
    post_id             Int64,
    author_profile_id   Int64,
    community_id        Nullable(Int64),
    event_type          LowCardinality(String),
    source              LowCardinality(String),
    dwell_ms            UInt32,
    position            UInt16,
    request_id          String
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(event_date)
ORDER BY (profile_id, event_type, event_time, post_id)
TTL event_date + INTERVAL 180 DAY;

CREATE TABLE IF NOT EXISTS post_snapshot
(
    post_id           Int64,
    author_profile_id Int64,
    community_id      Nullable(Int64),
    is_public_demo    Bool,
    is_active         Bool,
    allow_comments    Bool,
    created_at        DateTime64(3),
    text_length       UInt16,
    has_media         Bool,
    version           UInt64
)
ENGINE = ReplacingMergeTree(version)
ORDER BY post_id;

CREATE TABLE IF NOT EXISTS user_recent_viewed
(
    profile_id Int64,
    post_id    Int64,
    viewed_at  DateTime64(3)
)
ENGINE = ReplacingMergeTree(viewed_at)
ORDER BY (profile_id, post_id)
TTL toDate(viewed_at) + INTERVAL 30 DAY;

CREATE TABLE IF NOT EXISTS post_stats_1d
(
    post_id  Int64,
    day      Date,
    views    Int64,
    likes    Int64,
    comments Int64,
    reposts  Int64,
    hides    Int64,
    reports  Int64
)
ENGINE = SummingMergeTree
ORDER BY (day, post_id);

CREATE TABLE IF NOT EXISTS user_author_affinity_1d
(
    profile_id        Int64,
    author_profile_id Int64,
    day               Date,
    score             Int64
)
ENGINE = SummingMergeTree
ORDER BY (profile_id, author_profile_id, day);

CREATE TABLE IF NOT EXISTS user_community_affinity_1d
(
    profile_id   Int64,
    community_id Int64,
    day          Date,
    score        Int64
)
ENGINE = SummingMergeTree
ORDER BY (profile_id, community_id, day);

-- Materialized Views

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_post_stats
TO post_stats_1d AS
SELECT
    post_id,
    toDate(event_time)                          AS day,
    countIf(event_type = 'post_view')           AS views,
    countIf(event_type = 'post_like')
        - countIf(event_type = 'post_unlike')   AS likes,
    countIf(event_type = 'post_comment')        AS comments,
    countIf(event_type = 'post_repost')         AS reposts,
    countIf(event_type = 'post_hide')           AS hides,
    countIf(event_type = 'post_report')         AS reports
FROM post_events
GROUP BY post_id, day;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_author_affinity
TO user_author_affinity_1d AS
SELECT
    profile_id,
    author_profile_id,
    toDate(event_time) AS day,
    toInt64(sum(multiIf(
        event_type = 'post_view',    200,
        event_type = 'post_like',    3000,
        event_type = 'post_unlike', -3000,
        event_type = 'post_comment', 5000,
        event_type = 'post_repost',  6000,
        event_type = 'post_hide',   -8000,
        event_type = 'post_report',-20000,
        0
    ))) AS score
FROM post_events
WHERE profile_id > 0 AND author_profile_id > 0
GROUP BY profile_id, author_profile_id, day;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_community_affinity
TO user_community_affinity_1d AS
SELECT
    profile_id,
    community_id,
    toDate(event_time) AS day,
    toInt64(sum(multiIf(
        event_type = 'post_view',    200,
        event_type = 'post_like',    3000,
        event_type = 'post_unlike', -3000,
        event_type = 'post_comment', 5000,
        event_type = 'post_repost',  6000,
        event_type = 'post_hide',   -8000,
        event_type = 'post_report',-20000,
        0
    ))) AS score
FROM post_events
WHERE community_id IS NOT NULL
GROUP BY profile_id, community_id, day;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_recent_viewed
TO user_recent_viewed AS
SELECT
    profile_id,
    post_id,
    event_time AS viewed_at
FROM post_events
WHERE event_type IN ('post_view', 'post_hide', 'post_report');
