-- Add monitoring-engine columns to existing monitor_blocks table
ALTER TABLE monitor_blocks ADD COLUMN IF NOT EXISTS case_id UUID REFERENCES cases(id) ON DELETE CASCADE;
ALTER TABLE monitor_blocks ADD COLUMN IF NOT EXISTS last_executed_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_monitor_blocks_case_id ON monitor_blocks(case_id);

-- Add DSL polling toggle to cases
ALTER TABLE cases ADD COLUMN IF NOT EXISTS dsl_polling_enabled BOOLEAN NOT NULL DEFAULT true;
