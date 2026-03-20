-- Add MONITOR_RESULT to timeline_event_type enum
ALTER TYPE timeline_event_type ADD VALUE IF NOT EXISTS 'MONITOR_RESULT';

-- Add missing columns to cases
ALTER TABLE cases ADD COLUMN IF NOT EXISTS symbol_name TEXT NOT NULL DEFAULT '';
ALTER TABLE cases ADD COLUMN IF NOT EXISTS sector TEXT;

-- Add day_offset to timeline_events
ALTER TABLE timeline_events ADD COLUMN IF NOT EXISTS day_offset INT NOT NULL DEFAULT 0;
