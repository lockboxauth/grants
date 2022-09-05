-- +migrate Up
ALTER TABLE grants ADD COLUMN ancestor_ids VARCHAR[] NOT NULL DEFAULT array[]::varchar[],
		   ADD COLUMN revoked BOOLEAN NOT NULL DEFAULT false;

-- +migrate Down
ALTER TABLE grants DROP COLUMN IF EXISTS ancestor_ids,
		   DROP COLUMN IF EXISTS revoked;
