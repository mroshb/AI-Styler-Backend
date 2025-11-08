-- Remove legacy/unused tables and their dependent objects to keep schema lean

BEGIN;

DROP TABLE IF EXISTS rate_limits CASCADE;
DROP TABLE IF EXISTS image_quota_tracking CASCADE;
DROP TABLE IF EXISTS vendor_image_quotas CASCADE;
DROP TABLE IF EXISTS storage_backups CASCADE;
DROP TABLE IF EXISTS storage_health_checks CASCADE;
DROP TABLE IF EXISTS storage_metrics CASCADE;
DROP TABLE IF EXISTS user_plan_history CASCADE;
DROP TABLE IF EXISTS worker_stats CASCADE;

COMMIT;

