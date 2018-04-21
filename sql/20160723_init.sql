-- +migrate Up
CREATE TABLE grants (
	id CHAR(36) PRIMARY KEY,
	source_type TEXT NOT NULL DEFAULT '',
	source_id TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	scopes VARCHAR[] NOT NULL DEFAULT array[]::varchar[],
	profile_id VARCHAR(36) NOT NULL DEFAULT '',
	client_id VARCHAR(36) NOT NULL DEFAULT '',
	ip INET NOT NULL DEFAULT '0.0.0.0',
	used BOOLEAN NOT NULL DEFAULT false,

	UNIQUE(source_type, source_id)
);

-- +migrate Down
DROP TABLE grants;
