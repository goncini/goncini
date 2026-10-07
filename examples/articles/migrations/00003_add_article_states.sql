-- +goose Up
-- The articles there are were published; the app starts new ones as drafts.
ALTER TABLE articles ADD COLUMN state TEXT NOT NULL DEFAULT 'published';

-- +goose Down
ALTER TABLE articles DROP COLUMN state;
