-- Pipeline Builder schema migration
-- Evolves V1 JSONB-based schema to normalized relational structure

-- 1. agent_blocks ALTER: add AgentBlockPrompt structured fields
ALTER TABLE agent_blocks
  ADD COLUMN IF NOT EXISTS objective TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS input_desc TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS tools TEXT[] DEFAULT '{}',
  ADD COLUMN IF NOT EXISTS output_format TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS constraints TEXT,
  ADD COLUMN IF NOT EXISTS examples TEXT,
  ADD COLUMN IF NOT EXISTS stage_id UUID,
  ADD COLUMN IF NOT EXISTS is_template BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS template_id UUID,
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- 2. pipelines ALTER: make scripts nullable, drop JSONB columns
ALTER TABLE pipelines
  ALTER COLUMN success_script DROP NOT NULL,
  ALTER COLUMN success_script DROP DEFAULT,
  ALTER COLUMN failure_script DROP NOT NULL,
  ALTER COLUMN failure_script DROP DEFAULT;
ALTER TABLE pipelines DROP COLUMN IF EXISTS analysis_stages;
ALTER TABLE pipelines DROP COLUMN IF EXISTS monitors;

-- 3. price_alerts ALTER: make case_id nullable for pipeline-level alerts
ALTER TABLE price_alerts ALTER COLUMN case_id DROP NOT NULL;
-- Add pipeline_id index if needed
CREATE INDEX IF NOT EXISTS idx_price_alerts_pipeline_id ON price_alerts(pipeline_id);

-- 4. CREATE stages table
CREATE TABLE IF NOT EXISTS stages (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  pipeline_id     UUID NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
  section         TEXT NOT NULL CHECK (section IN ('analysis', 'monitoring', 'judgment')),
  order_index     INT NOT NULL DEFAULT 0,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_stages_pipeline_id ON stages(pipeline_id);

-- 5. Add FK from agent_blocks.stage_id to stages
ALTER TABLE agent_blocks
  ADD CONSTRAINT fk_agent_blocks_stage_id FOREIGN KEY (stage_id) REFERENCES stages(id) ON DELETE CASCADE;

-- 6. Add indexes for agent_blocks new columns
CREATE INDEX IF NOT EXISTS idx_agent_blocks_stage_id ON agent_blocks(stage_id);
CREATE INDEX IF NOT EXISTS idx_agent_blocks_template ON agent_blocks(is_template) WHERE is_template = true;

-- 7. CREATE monitor_blocks table
CREATE TABLE IF NOT EXISTS monitor_blocks (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  pipeline_id     UUID NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
  block_id        UUID NOT NULL UNIQUE REFERENCES agent_blocks(id) ON DELETE CASCADE,
  cron            TEXT NOT NULL,
  enabled         BOOLEAN NOT NULL DEFAULT true,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_monitor_blocks_pipeline_id ON monitor_blocks(pipeline_id);

-- 8. CREATE pipeline_jobs table
CREATE TABLE IF NOT EXISTS pipeline_jobs (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  pipeline_id     UUID NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
  symbol          TEXT NOT NULL,
  status          TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'RUNNING', 'COMPLETED', 'FAILED')),
  result          JSONB,
  error           TEXT,
  started_at      TIMESTAMPTZ,
  completed_at    TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_pipeline_jobs_pipeline_id ON pipeline_jobs(pipeline_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_jobs_status ON pipeline_jobs(status);

-- updated_at triggers for new tables
CREATE TRIGGER set_updated_at_monitor_blocks BEFORE UPDATE ON monitor_blocks FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
CREATE TRIGGER set_updated_at_agent_blocks BEFORE UPDATE ON agent_blocks FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
