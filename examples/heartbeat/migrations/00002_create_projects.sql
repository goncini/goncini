-- +goose Up
CREATE TABLE projects (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    key_hash   TEXT NOT NULL UNIQUE, -- the SHA-256 of its API key, in hex
    created_at BIGINT NOT NULL       -- Unix microseconds, as every time
);

-- +goose Down
DROP TABLE projects;
