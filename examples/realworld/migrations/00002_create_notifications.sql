-- +goose Up
CREATE TABLE notifications (
    id           INTEGER PRIMARY KEY,
    user_id      INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    article_slug TEXT NOT NULL,
    author_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at   INTEGER NOT NULL -- Unix microseconds
);
CREATE INDEX notifications_user ON notifications (user_id, created_at);

-- +goose Down
DROP TABLE notifications;
