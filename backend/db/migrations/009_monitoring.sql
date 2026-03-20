-- monitor_blocks table: stores individual monitoring blocks linked to cases and agent blocks
CREATE TABLE monitor_blocks (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  case_id          UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
  block_id         UUID NOT NULL REFERENCES agent_blocks(id),
  cron             TEXT NOT NULL,
  enabled          BOOLEAN NOT NULL DEFAULT true,
  last_executed_at TIMESTAMPTZ,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_monitor_blocks_case_id ON monitor_blocks(case_id);

-- Add DSL polling toggle to cases
ALTER TABLE cases ADD COLUMN dsl_polling_enabled BOOLEAN NOT NULL DEFAULT true;
