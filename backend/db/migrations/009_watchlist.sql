CREATE TABLE IF NOT EXISTS watchlist (
    id         BIGSERIAL PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    symbol     VARCHAR(20) NOT NULL,
    name       VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, symbol)
);

CREATE INDEX idx_watchlist_user_id ON watchlist(user_id);
