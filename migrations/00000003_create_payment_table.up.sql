CREATE TABLE payment (
    order_uid UUID PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
    transaction UUID NOT NULL UNIQUE,
    request_id UUID UNIQUE,
    currency CHAR(3) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    amount INT NOT NULL CHECK (amount > 0),
    payment_dt TIMESTAMPTZ NOT NULL CHECK (payment_dt > 0),
    bank VARCHAR(50) NOT NULL,
    delivery_cost INT NOT NULL CHECK (delivery_cost >= 0),
    goods_total INT NOT NULL CHECK (goods_total >= 0),
    custom_fee INT NOT NULL DEFAULT 0 CHECK (custom_fee >= 0)
);

CREATE INDEX idx_payment_transaction ON payment(transaction);