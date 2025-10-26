-- +goose Up
BEGIN;

-- 1) снять старый PK по transaction (если был)
ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_pkey;

-- 2) обеспечить уникальность transaction через индекс (идемпотентно)
CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_transaction_unique
    ON payments (transaction);

-- 3) убрать старый UNIQUE(order_uid), он больше не нужен
ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_order_uid_key;

-- 4) сделать order_uid NOT NULL и основным ключом
ALTER TABLE payments
    ALTER COLUMN order_uid SET NOT NULL;

ALTER TABLE payments
    ADD CONSTRAINT payments_pkey PRIMARY KEY (order_uid);

COMMIT;

-- +goose Down
BEGIN;

ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_pkey;

-- вернуть уникальность по order_uid
ALTER TABLE payments
    ADD CONSTRAINT payments_order_uid_key UNIQUE (order_uid);

-- (по желанию) убрать уникальный индекс по transaction
DROP INDEX IF EXISTS idx_payments_transaction_unique;

-- вернуть PK по transaction
ALTER TABLE payments
    ADD CONSTRAINT payments_pkey PRIMARY KEY (transaction);

COMMIT;
