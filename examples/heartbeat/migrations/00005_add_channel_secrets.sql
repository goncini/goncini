-- +goose Up
-- The secret that signs a webhook channel's webhooks: empty for email.
ALTER TABLE channels ADD COLUMN secret TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE channels DROP COLUMN secret;
