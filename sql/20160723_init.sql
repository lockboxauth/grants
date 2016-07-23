-- +migrate Up
CREATE TABLE grants (
	id CHAR(36) PRIMARY KEY,
	source_type TEXT NOT NULL,
	source_id TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL,
	scopes VARCHAR[] NOT NULL,
	profile_id VARCHAR(36) NOT NULL,
	client_id VARCHAR(36) NOT NULL,
	ip INET NOT NULL,
	used BOOLEAN NOT NULL,

	UNIQUE(source_type, source_id)
);

-- +migrate Down
DROP TABLE grants;
