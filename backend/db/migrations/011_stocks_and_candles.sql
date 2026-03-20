-- 011_stocks_and_candles.sql
-- Stock price data tables migrated from hack-the-invest (chartinglens)

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS stocks (
    symbol VARCHAR(10) PRIMARY KEY,
    name VARCHAR(100),
    market VARCHAR(10),
    updated_at DATE,
    category VARCHAR(10),
    backfill_done BOOLEAN
);

CREATE TABLE IF NOT EXISTS daily_candles (
    symbol VARCHAR(10) NOT NULL,
    date DATE NOT NULL,
    open BIGINT,
    high BIGINT,
    low BIGINT,
    close BIGINT,
    volume BIGINT,
    PRIMARY KEY (symbol, date)
);

-- Trigram index for fuzzy name search
CREATE INDEX IF NOT EXISTS idx_stocks_name_trgm ON stocks USING gin (name gin_trgm_ops);

-- Composite index for candle lookups (descending date for recent-first queries)
CREATE INDEX IF NOT EXISTS idx_daily_candles_symbol_date ON daily_candles (symbol, date DESC);
