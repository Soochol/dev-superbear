CREATE TABLE IF NOT EXISTS search_presets (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL,
    name       VARCHAR(255) NOT NULL,
    dsl        TEXT NOT NULL,
    nl_query   TEXT,
    is_public  BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_search_presets_user_id ON search_presets(user_id);
