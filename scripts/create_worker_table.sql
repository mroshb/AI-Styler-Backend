-- Quick fix: Create worker_jobs table if it doesn't exist
-- This is a standalone script to create the worker_jobs table
-- Run this directly if migration 0014_worker_service.sql hasn't been applied yet:
-- psql -d your_database -f scripts/create_worker_table.sql

BEGIN;

-- Enable required extension for UUID generation if not exists
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Create worker_jobs table if it doesn't exist
CREATE TABLE IF NOT EXISTS worker_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type TEXT NOT NULL,
    conversion_id UUID,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    priority INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled')),
    worker_id TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    error_message TEXT,
    payload JSONB,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for worker_jobs table (only if they don't exist)
CREATE INDEX IF NOT EXISTS idx_worker_jobs_status ON worker_jobs(status);
CREATE INDEX IF NOT EXISTS idx_worker_jobs_priority ON worker_jobs(priority DESC, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_worker_jobs_worker_id ON worker_jobs(worker_id);
CREATE INDEX IF NOT EXISTS idx_worker_jobs_conversion_id ON worker_jobs(conversion_id);
CREATE INDEX IF NOT EXISTS idx_worker_jobs_user_id ON worker_jobs(user_id);
CREATE INDEX IF NOT EXISTS idx_worker_jobs_created_at ON worker_jobs(created_at);

-- Create set_updated_at function if it doesn't exist
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Add trigger for worker_jobs updated_at (only if it doesn't exist)
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_worker_jobs_updated_at') THEN
        CREATE TRIGGER trg_worker_jobs_updated_at
        BEFORE UPDATE ON worker_jobs
        FOR EACH ROW EXECUTE FUNCTION set_updated_at();
    END IF;
END $$;

-- Create worker_stats table if it doesn't exist
CREATE TABLE IF NOT EXISTS worker_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    worker_id TEXT NOT NULL,
    jobs_processed INTEGER NOT NULL DEFAULT 0,
    jobs_failed INTEGER NOT NULL DEFAULT 0,
    avg_processing_time_ms INTEGER,
    last_heartbeat TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for worker_stats table
CREATE INDEX IF NOT EXISTS idx_worker_stats_worker_id ON worker_stats(worker_id);
CREATE INDEX IF NOT EXISTS idx_worker_stats_last_heartbeat ON worker_stats(last_heartbeat);

-- Add trigger for worker_stats updated_at (only if it doesn't exist)
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_worker_stats_updated_at') THEN
        CREATE TRIGGER trg_worker_stats_updated_at
        BEFORE UPDATE ON worker_stats
        FOR EACH ROW EXECUTE FUNCTION set_updated_at();
    END IF;
END $$;

COMMIT;

