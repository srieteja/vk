-- Add missing columns to calls table that may not exist in older schemas

ALTER TABLE calls ADD COLUMN IF NOT EXISTS scheduled_at TIMESTAMP;
ALTER TABLE calls ADD COLUMN IF NOT EXISTS started_at TIMESTAMP;
ALTER TABLE calls ADD COLUMN IF NOT EXISTS ended_at TIMESTAMP;
