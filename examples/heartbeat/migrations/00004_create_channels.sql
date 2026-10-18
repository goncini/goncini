-- +goose Up
CREATE TABLE channels (
    id         TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    kind       TEXT NOT NULL, -- email or webhook
    target     TEXT NOT NULL, -- the address, or the URL
    created_at BIGINT NOT NULL
);
CREATE INDEX channels_project ON channels (project_id, created_at);

-- The alerts sent to each channel, so that an alert handled twice isn't
-- sent twice.
CREATE TABLE deliveries (
    flip_id    TEXT NOT NULL,
    channel_id TEXT NOT NULL REFERENCES channels (id) ON DELETE CASCADE,
    created_at BIGINT NOT NULL,
    PRIMARY KEY (flip_id, channel_id)
);
CREATE INDEX deliveries_created_at ON deliveries (created_at);

-- +goose Down
DROP TABLE deliveries;
DROP TABLE channels;
