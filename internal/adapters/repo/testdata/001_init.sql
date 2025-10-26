
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS orders (
                                      order_uid           TEXT        PRIMARY KEY,
                                      track_number        TEXT        NOT NULL,
                                      entry               TEXT        NOT NULL,
                                      locale              TEXT        NOT NULL,
                                      internal_signature  TEXT        NOT NULL,
                                      customer_id         TEXT        NOT NULL,
                                      delivery_service    TEXT        NOT NULL,
                                      shard_key           TEXT        NOT NULL,
                                      sm_id               BIGINT      NOT NULL,
                                      date_created        TIMESTAMPTZ NOT NULL DEFAULT now(),
    oof_shard           BIGINT      NOT NULL
    );

CREATE TABLE IF NOT EXISTS deliveries (
                                          order_uid     TEXT PRIMARY KEY
                                          REFERENCES orders(order_uid) ON UPDATE CASCADE ON DELETE CASCADE,
    delivery_name TEXT NOT NULL,
    phone         TEXT NOT NULL,
    zip           TEXT NOT NULL,
    city          TEXT NOT NULL,
    address       TEXT NOT NULL,
    region        TEXT NOT NULL,
    email         TEXT NOT NULL
    );

CREATE TABLE IF NOT EXISTS payments (
                                        order_uid     TEXT PRIMARY KEY
                                        REFERENCES orders(order_uid) ON UPDATE CASCADE ON DELETE CASCADE,
    transaction   TEXT NOT NULL,
    request_id    TEXT NOT NULL,
    currency      TEXT NOT NULL,
    provider      TEXT NOT NULL,
    amount        BIGINT NOT NULL,
    payment_dt    TIMESTAMPTZ NOT NULL,
    bank          TEXT NOT NULL,
    delivery_cost BIGINT NOT NULL,
    goods_total   BIGINT NOT NULL,
    custom_fee    BIGINT NOT NULL,
    CONSTRAINT chk_amount_nonneg        CHECK (amount >= 0),
    CONSTRAINT chk_delivery_cost_nonneg CHECK (delivery_cost >= 0),
    CONSTRAINT chk_goods_total_nonneg   CHECK (goods_total >= 0),
    CONSTRAINT chk_custom_fee_nonneg    CHECK (custom_fee >= 0),
    CONSTRAINT chk_currency_iso3        CHECK (currency ~ '^[A-Z]{3}$')
    );

CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_transaction_unique
    ON payments (transaction);

CREATE TABLE IF NOT EXISTS order_items (
                                           order_uid    TEXT   NOT NULL
                                           REFERENCES orders(order_uid) ON UPDATE CASCADE ON DELETE CASCADE,

    chrt_id      TEXT   NOT NULL,
    track_number TEXT   NOT NULL,
    price        BIGINT NOT NULL,
    rid          TEXT   NOT NULL,

    item_name    TEXT   NOT NULL,
    sale         BIGINT NOT NULL DEFAULT 0,
    item_size    BIGINT NOT NULL,
    total_price  BIGINT NOT NULL,

    nm_id        TEXT   NOT NULL,
    brand        TEXT   NOT NULL,
    status       BIGINT NOT NULL,

    CONSTRAINT pk_order_items PRIMARY KEY (order_uid, chrt_id),
    CONSTRAINT chk_item_price_nonneg       CHECK (price >= 0),
    CONSTRAINT chk_item_sale_nonneg        CHECK (sale  >= 0),
    CONSTRAINT chk_item_total_price_nonneg CHECK (total_price >= 0),
    CONSTRAINT chk_item_status_nonneg      CHECK (status >= 0)
    );

CREATE TABLE IF NOT EXISTS consumer_offsets (
                                                topic         TEXT NOT NULL,
                                                "partition"   INT  NOT NULL,
                                                group_id      TEXT NOT NULL,
                                                "offset"      BIGINT NOT NULL,
                                                processed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (topic, partition, group_id)
    );

CREATE INDEX IF NOT EXISTS idx_orders_date_created ON orders (date_created DESC);
CREATE INDEX IF NOT EXISTS idx_orders_track_number ON orders (track_number);
CREATE INDEX IF NOT EXISTS idx_orders_customer_id  ON orders (customer_id);

CREATE INDEX IF NOT EXISTS idx_payments_provider ON payments (provider);
CREATE INDEX IF NOT EXISTS idx_payments_currency ON payments (currency);

CREATE INDEX IF NOT EXISTS idx_items_track_number ON order_items (track_number);

CREATE INDEX IF NOT EXISTS idx_orders_track_trgm ON orders USING gin (track_number gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_orders_uid_trgm   ON orders USING gin (order_uid    gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_payments_tx_trgm  ON payments USING gin (transaction gin_trgm_ops);
