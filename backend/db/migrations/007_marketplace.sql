-- Marketplace tables for sharing and forking pipelines, agent blocks, search presets, and judgment scripts

CREATE TYPE marketplace_item_type AS ENUM ('PIPELINE', 'AGENT_BLOCK', 'SEARCH_PRESET', 'JUDGMENT_SCRIPT');
CREATE TYPE marketplace_status AS ENUM ('ACTIVE', 'HIDDEN', 'REMOVED');
CREATE TYPE usage_action AS ENUM ('VIEW', 'FORK', 'EXECUTE', 'LIKE');

-- 마켓플레이스 아이템 (4가지 공유 타입 통합)
CREATE TABLE marketplace_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  type marketplace_item_type NOT NULL,
  title TEXT NOT NULL,
  description TEXT NOT NULL,
  tags TEXT[] DEFAULT '{}',

  -- 원본 참조 (각 타입별 1:1)
  pipeline_id UUID UNIQUE REFERENCES pipelines(id),
  agent_block_id UUID UNIQUE REFERENCES agent_blocks(id),
  search_preset_id UUID UNIQUE REFERENCES search_presets(id),
  judgment_script_id UUID UNIQUE,

  -- Fork 추적
  forked_from_id UUID REFERENCES marketplace_items(id),
  fork_count INT NOT NULL DEFAULT 0,

  -- 통계
  usage_count INT NOT NULL DEFAULT 0,
  view_count INT NOT NULL DEFAULT 0,
  like_count INT NOT NULL DEFAULT 0,

  -- 백테스트 성과 (Verified 뱃지 기준)
  verified BOOLEAN NOT NULL DEFAULT false,
  backtest_job_id UUID REFERENCES backtest_jobs(id),
  backtest_win_rate NUMERIC(5,2),
  backtest_avg_return NUMERIC(8,2),
  backtest_total_events INT,

  -- 상태
  status marketplace_status NOT NULL DEFAULT 'ACTIVE',
  published_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  -- Full-text search vector (title + description + tags)
  search_vector TSVECTOR
);

-- 인덱스
CREATE INDEX idx_marketplace_items_type_status ON marketplace_items(type, status);
CREATE INDEX idx_marketplace_items_usage_count ON marketplace_items(usage_count);
CREATE INDEX idx_marketplace_items_win_rate ON marketplace_items(backtest_win_rate);
CREATE INDEX idx_marketplace_items_published_at ON marketplace_items(published_at);
CREATE INDEX idx_marketplace_items_fork_count ON marketplace_items(fork_count);
CREATE INDEX idx_marketplace_items_like_count ON marketplace_items(like_count);
CREATE INDEX idx_marketplace_search ON marketplace_items USING GIN (search_vector);

-- search_vector 자동 갱신 트리거
CREATE OR REPLACE FUNCTION marketplace_search_vector_update() RETURNS TRIGGER AS $$
BEGIN
  NEW.search_vector :=
    setweight(to_tsvector('simple', COALESCE(NEW.title, '')), 'A') ||
    setweight(to_tsvector('simple', COALESCE(NEW.description, '')), 'B') ||
    setweight(to_tsvector('simple', COALESCE(array_to_string(NEW.tags, ' '), '')), 'C');
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_marketplace_search_vector
  BEFORE INSERT OR UPDATE OF title, description, tags
  ON marketplace_items
  FOR EACH ROW
  EXECUTE FUNCTION marketplace_search_vector_update();

-- 좋아요 테이블
CREATE TABLE marketplace_likes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  item_id UUID NOT NULL REFERENCES marketplace_items(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, item_id)
);

CREATE INDEX idx_marketplace_likes_item ON marketplace_likes(item_id);

-- 사용량 로그 테이블
CREATE TABLE marketplace_usage_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  item_id UUID NOT NULL REFERENCES marketplace_items(id) ON DELETE CASCADE,
  action usage_action NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_marketplace_usage_logs_item_date ON marketplace_usage_logs(item_id, created_at);
CREATE INDEX idx_marketplace_usage_logs_dedup ON marketplace_usage_logs(item_id, user_id, action, created_at);

-- updated_at 자동 갱신
CREATE OR REPLACE FUNCTION marketplace_updated_at() RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_marketplace_updated_at
  BEFORE UPDATE ON marketplace_items
  FOR EACH ROW
  EXECUTE FUNCTION marketplace_updated_at();
