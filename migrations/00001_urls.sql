-- +goose Up
CREATE TABLE IF NOT EXISTS short_urls(
    url text NOT NULL PRIMARY KEY,
    shortened_url text NOT NULL,

    UNIQUE (url),
    UNIQUE (shortened_url)

);

CREATE INDEX IF NOT EXISTS short ON short_urls(shortened_url);
-- +goose Down
DROP TABLE IF EXISTS short_urls;
