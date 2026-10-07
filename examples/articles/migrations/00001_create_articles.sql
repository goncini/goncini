-- +goose Up
CREATE TABLE articles (
    slug       TEXT PRIMARY KEY,
    title      TEXT NOT NULL,
    body       TEXT NOT NULL,
    tags       TEXT NOT NULL, -- a JSON array
    created_at INTEGER NOT NULL -- Unix microseconds
);
CREATE INDEX articles_created_at ON articles (created_at);

-- +goose Down
DROP TABLE articles;
