-- BazaarPay Payment Service Schema Migration
-- Creates tables for BazaarPay payment gateway integration

BEGIN;

-- pre_payments table - ذخیره اطلاعات قبل از پرداخت
CREATE TABLE IF NOT EXISTS pre_payments (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_id VARCHAR(255) NOT NULL UNIQUE,
    segment VARCHAR(255) NOT NULL CHECK (segment IN ('plan', 'order', 'shop')),
    segment_id INTEGER NOT NULL,
    plan_id UUID REFERENCES payment_plans(id) ON DELETE SET NULL,
    amount BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pre_payments_user_id ON pre_payments(user_id);
CREATE INDEX IF NOT EXISTS idx_pre_payments_order_id ON pre_payments(order_id);
CREATE INDEX IF NOT EXISTS idx_pre_payments_segment ON pre_payments(segment, segment_id);

-- plan_payments table - ذخیره پرداخت‌های موفق پلن
CREATE TABLE IF NOT EXISTS plan_payments (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_id VARCHAR(255) NOT NULL UNIQUE,
    ref_number VARCHAR(255),
    amount BIGINT NOT NULL,
    card_number VARCHAR(19),
    status INTEGER NOT NULL,
    result INTEGER NOT NULL,
    message TEXT,
    description TEXT,
    segment VARCHAR(255) NOT NULL DEFAULT 'plan',
    segment_id INTEGER NOT NULL,
    paid_at VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_plan_payments_user_id ON plan_payments(user_id);
CREATE INDEX IF NOT EXISTS idx_plan_payments_order_id ON plan_payments(order_id);
CREATE INDEX IF NOT EXISTS idx_plan_payments_ref_number ON plan_payments(ref_number);
CREATE INDEX IF NOT EXISTS idx_plan_payments_created_at ON plan_payments(created_at DESC);

-- order_payments table - ذخیره پرداخت‌های موفق سفارش
CREATE TABLE IF NOT EXISTS order_payments (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_id VARCHAR(255) NOT NULL UNIQUE,
    ref_number VARCHAR(255),
    amount BIGINT NOT NULL,
    card_number VARCHAR(19),
    status INTEGER NOT NULL,
    result INTEGER NOT NULL,
    message TEXT,
    description TEXT,
    segment VARCHAR(255) NOT NULL DEFAULT 'order',
    segment_id INTEGER NOT NULL,
    paid_at VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_order_payments_user_id ON order_payments(user_id);
CREATE INDEX IF NOT EXISTS idx_order_payments_order_id ON order_payments(order_id);
CREATE INDEX IF NOT EXISTS idx_order_payments_ref_number ON order_payments(ref_number);
CREATE INDEX IF NOT EXISTS idx_order_payments_created_at ON order_payments(created_at DESC);

-- shop_payments table - ذخیره پرداخت‌های موفق خرید مدل
CREATE TABLE IF NOT EXISTS shop_payments (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_id VARCHAR(255) NOT NULL UNIQUE,
    ref_number VARCHAR(255),
    amount BIGINT NOT NULL,
    card_number VARCHAR(19),
    status INTEGER NOT NULL,
    result INTEGER NOT NULL,
    message TEXT,
    description TEXT,
    segment VARCHAR(255) NOT NULL DEFAULT 'shop',
    segment_id INTEGER NOT NULL,
    paid_at VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_shop_payments_user_id ON shop_payments(user_id);
CREATE INDEX IF NOT EXISTS idx_shop_payments_order_id ON shop_payments(order_id);
CREATE INDEX IF NOT EXISTS idx_shop_payments_ref_number ON shop_payments(ref_number);
CREATE INDEX IF NOT EXISTS idx_shop_payments_created_at ON shop_payments(created_at DESC);

COMMIT;

