-- +goose Up

ALTER TABLE short_urls
    ADD COLUMN expires_at TIMESTAMPTZ,
    ADD COLUMN max_clicks INTEGER,
    ADD COLUMN clicks INTEGER NOT NULL DEFAULT 0;

ALTER TABLE short_urls
    ADD CONSTRAINT short_urls_max_clicks_check
        CHECK (max_clicks IS NULL OR max_clicks > 0);


-- +goose Down

ALTER TABLE short_urls
    DROP CONSTRAINT short_urls_max_clicks_check;

ALTER TABLE short_urls
    DROP COLUMN expires_at,
    DROP COLUMN max_clicks,
    DROP COLUMN clicks;