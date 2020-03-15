-- +migrate Up
ALTER TABLE grants ADD COLUMN account_id VARCHAR(36) NOT NULL DEFAULT '';

-- +migrate Down
ALTER TABLE grants DROP COLUMN IF EXISTS account_id;
