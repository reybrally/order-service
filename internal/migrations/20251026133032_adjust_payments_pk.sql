-- +goose Up
BEGIN;

ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_pkey;

CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_transaction_unique
    ON payments (transaction);

ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_order_uid_key;

ALTER TABLE payments
    ALTER COLUMN order_uid SET NOT NULL;

ALTER TABLE payments
    ADD CONSTRAINT payments_pkey PRIMARY KEY (order_uid);

COMMIT;

-- +goose Down
BEGIN;

ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_pkey;

ALTER TABLE payments
    ADD CONSTRAINT payments_order_uid_key UNIQUE (order_uid);

DROP INDEX IF EXISTS idx_payments_transaction_unique;

ALTER TABLE payments
    ADD CONSTRAINT payments_pkey PRIMARY KEY (transaction);

COMMIT;
