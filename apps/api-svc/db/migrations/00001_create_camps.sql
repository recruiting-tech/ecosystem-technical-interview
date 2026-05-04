-- +goose Up
CREATE TABLE camps (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT NOT NULL,
    sport        TEXT NOT NULL,
    location     TEXT NOT NULL,
    capacity     INT  NOT NULL,
    start_date   DATE NOT NULL,
    end_date     DATE NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_camps_sport ON camps (sport);
CREATE INDEX idx_camps_start_date ON camps (start_date);

-- +goose Down
DROP TABLE camps;
