-- Real payment-gateway integration: server-derived integer-cents amounts,
-- gateway correlation, and idempotent webhook delivery.

ALTER TABLE payments ADD COLUMN IF NOT EXISTS amount_cents BIGINT NOT NULL DEFAULT 0;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS currency VARCHAR(3) NOT NULL DEFAULT 'usd';
ALTER TABLE payments ADD COLUMN IF NOT EXISTS gateway_payment_id VARCHAR(255);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS duration_minutes INTEGER NOT NULL DEFAULT 0;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS advocate_commission_cents BIGINT NOT NULL DEFAULT 0;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS platform_fee_cents BIGINT NOT NULL DEFAULT 0;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS refunded_amount_cents BIGINT NOT NULL DEFAULT 0;

-- Old float `amount`/`advocate_commission`/`platform_fee` columns are left
-- in place (unmanaged by the Go model going forward) rather than dropped,
-- since dropping is irreversible and these are cheap to leave dangling.
-- `amount` has no default and is NOT NULL though, and new inserts no
-- longer populate it, so it must stop being required.
ALTER TABLE payments ALTER COLUMN amount DROP NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_gateway_payment_id
    ON payments(payment_gateway, gateway_payment_id)
    WHERE gateway_payment_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS webhook_events (
    id SERIAL PRIMARY KEY,
    provider VARCHAR(50) NOT NULL,
    event_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider, event_id)
);
