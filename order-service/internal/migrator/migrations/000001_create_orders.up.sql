CREATE TABLE IF NOT EXISTS orders (
    id         BIGSERIAL   PRIMARY KEY,
    user_id    BIGINT      NOT NULL,
    status     VARCHAR(20) NOT NULL DEFAULT 'new'
                           CHECK (status IN ('new', 'confirmed', 'cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_orders_user_id ON orders (user_id);
CREATE INDEX idx_orders_status  ON orders (status);
