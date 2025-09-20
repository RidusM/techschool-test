CREATE EXTENSION IF NOT EXISTS "pgcrypto" SCHEMA public;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp" SCHEMA public;

CREATE TABLE orders (
    order_uid UUID PRIMARY KEY,
    track_number VARCHAR(50) NOT NULL UNIQUE,
    entry VARCHAR(10) NOT NULL,
    locale CHAR(2) NOT NULL,
    internal_signature VARCHAR(255),
    customer_id VARCHAR(50) NOT NULL,
    delivery_service VARCHAR(50) NOT NULL,
    shardkey VARCHAR(10) NOT NULL,
    sm_id INT NOT NULL,
    date_created TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    oof_shard CHAR(1) NOT NULL
);

CREATE INDEX idx_orders_date_created ON orders(date_created DESC);
CREATE INDEX idx_orders_track_number ON orders(track_number);
CREATE INDEX idx_orders_customer_id ON orders(customer_id);