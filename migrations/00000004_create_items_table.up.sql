CREATE TABLE items (
    items_id UUID PRIMARY KEY,
    order_uid UUID NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
    chrt_id BIGINT NOT NULL,
    track_number VARCHAR(50) NOT NULL,
    price INT NOT NULL CHECK (price > 0),
    rid UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    sale INT NOT NULL DEFAULT 0 CHECK (sale BETWEEN 0 AND 100),
    size VARCHAR(10) NOT NULL,
    total_price INT NOT NULL CHECK (total_price > 0),
    nm_id BIGINT NOT NULL,
    brand VARCHAR(100) NOT NULL,
    status SMALLINT NOT NULL CHECK (status >= 0)
);

CREATE INDEX idx_items_order_uid ON items(order_uid);