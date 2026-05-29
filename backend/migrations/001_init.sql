-- Initial schema for flash sale reservation system

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    total_quantity INT NOT NULL CHECK (total_quantity >= 0),
    reserved_quantity INT NOT NULL DEFAULT 0 CHECK (reserved_quantity >= 0),
    CONSTRAINT items_reserved_lte_total CHECK (reserved_quantity <= total_quantity)
);

CREATE TABLE IF NOT EXISTS reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id UUID NOT NULL REFERENCES items(id),
    user_id TEXT NOT NULL,
    quantity INT NOT NULL CHECK (quantity > 0),
    status TEXT NOT NULL CHECK (status IN ('active', 'released', 'expired')),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_reservations_user_status ON reservations(user_id, status);
CREATE INDEX IF NOT EXISTS idx_reservations_status_expires ON reservations(status, expires_at);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    key TEXT PRIMARY KEY,
    request_hash TEXT NOT NULL,
    response_status INT NOT NULL,
    response_body JSONB NOT NULL,
    reservation_id UUID REFERENCES reservations(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
