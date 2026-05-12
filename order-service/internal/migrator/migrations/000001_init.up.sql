-- Orders + items: operational tables for the order-service domain.
CREATE TABLE IF NOT EXISTS orders (
    id         UUID        PRIMARY KEY,
    user_id    UUID        NOT NULL,
    status     VARCHAR(20) NOT NULL DEFAULT 'new'
                           CHECK (status IN ('new', 'confirmed', 'cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_orders_user_id ON orders (user_id);
CREATE INDEX idx_orders_status  ON orders (status);

CREATE TABLE IF NOT EXISTS order_items (
    id         UUID        PRIMARY KEY,
    order_id   UUID        NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id UUID        NOT NULL,
    quantity   INT         NOT NULL CHECK (quantity > 0),
    price      BIGINT      NOT NULL CHECK (price > 0), -- cents
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_order_items_order_id ON order_items (order_id);

-- Outbox: events written transactionally with order changes, drained by the relay.
--   sent_at NULL AND dead_at NULL → pending
--   sent_at NOT NULL              → delivered to Kafka
--   dead_at NOT NULL              → exceeded max attempts, kept for forensics
--
-- id is a UUID v7 generated in code — time-sortable, so ORDER BY id is still FIFO.
CREATE TABLE IF NOT EXISTS outbox (
    id         UUID         PRIMARY KEY,
    topic      VARCHAR(255) NOT NULL,
    key        TEXT         NOT NULL DEFAULT '',   -- Kafka partition key (order_id)
    payload    JSONB        NOT NULL,
    metadata   JSONB,                              -- request_id, trace context, …
    attempts   INT          NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    sent_at    TIMESTAMPTZ,
    dead_at    TIMESTAMPTZ
);

-- Partial index covering only rows the relay polls for, ordered by primary key.
CREATE INDEX idx_outbox_unsent ON outbox (id) WHERE sent_at IS NULL AND dead_at IS NULL;
