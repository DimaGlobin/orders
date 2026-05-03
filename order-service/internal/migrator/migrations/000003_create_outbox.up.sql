CREATE TABLE IF NOT EXISTS outbox (
    id         BIGSERIAL    PRIMARY KEY,
    topic      VARCHAR(255) NOT NULL,
    key        TEXT         NOT NULL DEFAULT '', -- Kafka partition key (e.g. order_id)
    payload    JSONB        NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    sent_at    TIMESTAMPTZ                       -- NULL = pending, NOT NULL = delivered
);

-- Partial index: relay queries only unsent events
CREATE INDEX idx_outbox_unsent ON outbox (created_at ASC) WHERE sent_at IS NULL;
