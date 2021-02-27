package postgres

import (
	"database/sql"
	"embed"

	migrate "github.com/rubenv/sql-migrate"
)

//go:embed sql/*
var migrations embed.FS

// ApplyMigrations runs the necessary database migrations to make the database
// match the expected schema against the passed connection.
func ApplyMigrations(connection *sql.DB, direction migrate.MigrationDirection) error {
	migrations := &migrate.EmbedFileSystemMigrationSource{
		FileSystem: migrations,
		Root:       "sql",
	}
	_, err := migrate.Exec(connection, "postgres", migrations, direction)
	return err
}
