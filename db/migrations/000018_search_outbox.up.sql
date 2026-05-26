CREATE TABLE IF NOT EXISTS search_outbox (
    id          BIGSERIAL   PRIMARY KEY,
    entity_type TEXT        NOT NULL CHECK (entity_type IN ('user', 'community', 'post')),
    entity_id   BIGINT      NOT NULL,
    operation   TEXT        NOT NULL CHECK (operation IN ('upsert', 'delete')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS indexer_state (
    id                INT    PRIMARY KEY DEFAULT 1,
    last_processed_id BIGINT NOT NULL    DEFAULT 0,
    CONSTRAINT indexer_state_single_row CHECK (id = 1)
);

INSERT INTO indexer_state (id, last_processed_id)
VALUES (1, 0)
ON CONFLICT DO NOTHING;
