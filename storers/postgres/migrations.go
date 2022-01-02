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
	migrations := MigrationsSource()
	_, err := migrate.Exec(connection, "postgres", migrations, direction)
	return err
}

// MigrationsSource returns a migrate.MigrationSource to apply the migrations
// for this storer.
func MigrationsSource() migrate.MigrationSource {
	return &migrate.EmbedFileSystemMigrationSource{
		FileSystem: migrations,
		Root:       "sql",
	}
}
