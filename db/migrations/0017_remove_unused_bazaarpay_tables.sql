-- Remove unused BazaarPay tables that are no longer referenced by the application
-- This migration is safe to run multiple times thanks to IF EXISTS guards.

BEGIN;

-- Drop order payments tracking table if it exists
DROP TABLE IF EXISTS order_payments CASCADE;

-- Drop shop payments tracking table if it exists
DROP TABLE IF EXISTS shop_payments CASCADE;

COMMIT;

