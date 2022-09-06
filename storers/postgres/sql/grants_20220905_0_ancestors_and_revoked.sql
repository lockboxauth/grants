-- +migrate Up
CREATE TABLE grants_ancestors (
	grant_id VARCHAR NOT NULL,
	ancestor_id VARCHAR NOT NULL,

	UNIQUE(grant_id, ancestor_id)
);

ALTER TABLE grants ADD COLUMN revoked BOOLEAN NOT NULL DEFAULT false;

-- +migrate Down
ALTER TABLE grants DROP COLUMN IF EXISTS revoked;

DROP TABLE grants_ancestors;
