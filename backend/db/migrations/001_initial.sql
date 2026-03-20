-- Enum types
CREATE TYPE case_status AS ENUM ('LIVE', 'CLOSED_SUCCESS', 'CLOSED_FAILURE', 'BACKTEST');
CREATE TYPE timeline_event_type AS ENUM ('NEWS', 'DISCLOSURE', 'SECTOR', 'PRICE_ALERT', 'TRADE', 'PIPELINE_RESULT');
CREATE TYPE trade_type AS ENUM ('BUY', 'SELL');

-- users table
CREATE TABLE users (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email      TEXT NOT NULL UNIQUE,
  name       TEXT,
  image      TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- pipelines table
CREATE TABLE pipelines (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id          UUID NOT NULL REFERENCES users(id),
  name             TEXT NOT NULL,
  description      TEXT NOT NULL DEFAULT '',
  analysis_stages  JSONB NOT NULL DEFAULT '[]',
  monitors         JSONB NOT NULL DEFAULT '[]',
  success_script   TEXT NOT NULL DEFAULT '',
  failure_script   TEXT NOT NULL DEFAULT '',
  is_public        BOOLEAN NOT NULL DEFAULT false,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_pipelines_user_id ON pipelines(user_id);

-- cases table
CREATE TABLE cases (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id         UUID NOT NULL REFERENCES users(id),
  pipeline_id     UUID NOT NULL REFERENCES pipelines(id),
  symbol          TEXT NOT NULL,
  status          case_status NOT NULL DEFAULT 'LIVE',
  event_date      DATE NOT NULL,
  event_snapshot  JSONB NOT NULL,
  success_script  TEXT NOT NULL,
  failure_script  TEXT NOT NULL,
  closed_at       DATE,
  closed_reason   TEXT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_cases_user_id ON cases(user_id);
CREATE INDEX idx_cases_symbol ON cases(symbol);
CREATE INDEX idx_cases_status ON cases(status);

-- timeline_events table
CREATE TABLE timeline_events (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  case_id     UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
  date        DATE NOT NULL,
  type        timeline_event_type NOT NULL,
  title       TEXT NOT NULL,
  content     TEXT NOT NULL,
  ai_analysis TEXT,
  data        JSONB,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_timeline_events_case_id ON timeline_events(case_id);
CREATE INDEX idx_timeline_events_date ON timeline_events(date);

-- trades table
CREATE TABLE trades (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  case_id    UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
  user_id    UUID NOT NULL REFERENCES users(id),
  type       trade_type NOT NULL,
  price      DOUBLE PRECISION NOT NULL,
  quantity   INTEGER NOT NULL,
  fee        DOUBLE PRECISION NOT NULL DEFAULT 0,
  date       DATE NOT NULL,
  note       TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_trades_case_id ON trades(case_id);
CREATE INDEX idx_trades_user_id ON trades(user_id);

-- agent_blocks table
CREATE TABLE agent_blocks (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       UUID NOT NULL REFERENCES users(id),
  name          TEXT NOT NULL,
  instruction   TEXT NOT NULL,
  system_prompt TEXT,
  allowed_tools JSONB,
  output_schema JSONB,
  is_public     BOOLEAN NOT NULL DEFAULT false,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_agent_blocks_user_id ON agent_blocks(user_id);

-- price_alerts table
CREATE TABLE price_alerts (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  case_id      UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
  pipeline_id  UUID REFERENCES pipelines(id),
  condition    TEXT NOT NULL,
  label        TEXT NOT NULL,
  triggered    BOOLEAN NOT NULL DEFAULT false,
  triggered_at DATE,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_price_alerts_case_id ON price_alerts(case_id);

-- updated_at trigger
CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_updated_at_users BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
CREATE TRIGGER set_updated_at_pipelines BEFORE UPDATE ON pipelines FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
CREATE TRIGGER set_updated_at_cases BEFORE UPDATE ON cases FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
