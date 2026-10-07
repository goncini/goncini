-- +goose Up
ALTER TABLE articles ADD COLUMN id TEXT;
-- The articles there are get random UUIDs (version 4); the app gives new
-- ones time-ordered UUIDs (version 7).
UPDATE articles SET id = lower(
    hex(randomblob(4)) || '-' || hex(randomblob(2)) || '-4' || substr(hex(randomblob(2)), 2) || '-' ||
    substr('89ab', 1 + abs(random()) % 4, 1) || substr(hex(randomblob(2)), 2) || '-' || hex(randomblob(6)));
CREATE UNIQUE INDEX articles_id ON articles (id);

-- +goose Down
DROP INDEX articles_id;
ALTER TABLE articles DROP COLUMN id;
