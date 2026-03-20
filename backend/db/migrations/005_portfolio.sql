-- Portfolio tables for FIFO-based position tracking and realized PnL

CREATE TYPE market_type AS ENUM ('KR', 'US');

-- 포트폴리오 포지션 (종목별 보유 현황)
CREATE TABLE portfolio_positions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  symbol TEXT NOT NULL,
  symbol_name TEXT NOT NULL,
  market market_type NOT NULL DEFAULT 'KR',
  quantity INT NOT NULL DEFAULT 0,
  avg_cost_price NUMERIC(12,2) NOT NULL DEFAULT 0,
  total_cost NUMERIC(12,2) NOT NULL DEFAULT 0,
  sector TEXT,
  sector_name TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, symbol)
);

CREATE INDEX idx_portfolio_positions_user_id ON portfolio_positions(user_id);

-- FIFO 로트 (매수 단위별 잔여 수량 추적)
CREATE TABLE fifo_lots (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  position_id UUID NOT NULL REFERENCES portfolio_positions(id),
  trade_id UUID NOT NULL,
  buy_date TIMESTAMPTZ NOT NULL,
  buy_price NUMERIC(12,2) NOT NULL,
  original_qty INT NOT NULL,
  remaining_qty INT NOT NULL,
  fee NUMERIC(12,2) NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_fifo_lots_position_remaining ON fifo_lots(position_id, remaining_qty);

-- 실현 손익 레코드
CREATE TABLE realized_pnls (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  position_id UUID NOT NULL REFERENCES portfolio_positions(id),
  sell_trade_id UUID NOT NULL,
  buy_lot_id UUID NOT NULL,
  quantity INT NOT NULL,
  buy_price NUMERIC(12,2) NOT NULL,
  sell_price NUMERIC(12,2) NOT NULL,
  gross_pnl NUMERIC(12,2) NOT NULL,
  fee NUMERIC(12,2) NOT NULL,
  tax NUMERIC(12,2) NOT NULL,
  net_pnl NUMERIC(12,2) NOT NULL,
  realized_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_realized_pnls_position_date ON realized_pnls(position_id, realized_at);
