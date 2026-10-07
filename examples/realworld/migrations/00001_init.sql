-- +goose Up
CREATE TABLE users (
    id            INTEGER PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    bio           TEXT,
    image         TEXT
);

CREATE TABLE follows (
    follower_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    followed_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    PRIMARY KEY (follower_id, followed_id)
);

CREATE TABLE articles (
    id          INTEGER PRIMARY KEY,
    slug        TEXT NOT NULL UNIQUE,
    title       TEXT NOT NULL,
    description TEXT NOT NULL,
    body        TEXT NOT NULL,
    author_id   INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at  INTEGER NOT NULL, -- Unix microseconds
    updated_at  INTEGER NOT NULL
);
CREATE INDEX articles_created_at ON articles (created_at);

CREATE TABLE article_tags (
    article_id INTEGER NOT NULL REFERENCES articles (id) ON DELETE CASCADE,
    position   INTEGER NOT NULL,
    tag        TEXT NOT NULL,
    PRIMARY KEY (article_id, position)
);
CREATE INDEX article_tags_tag ON article_tags (tag);

CREATE TABLE favorites (
    user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    article_id INTEGER NOT NULL REFERENCES articles (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, article_id)
);

CREATE TABLE comments (
    id         INTEGER PRIMARY KEY,
    article_id INTEGER NOT NULL REFERENCES articles (id) ON DELETE CASCADE,
    author_id  INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    body       TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- +goose Down
DROP TABLE comments;
DROP TABLE favorites;
DROP TABLE article_tags;
DROP TABLE articles;
DROP TABLE follows;
DROP TABLE users;
