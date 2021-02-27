package postgres

import (
	"database/sql"
	"embed"

	migrate "github.com/rubenv/sql-migrate"
)

//go:embed sql/*
var migrations embed.FS

func ApplyMigrations(connection *sql.DB, direction migrate.MigrationDirection) error {
	migrations := &migrate.EmbedFileSystemMigrationSource{
		FileSystem: migrations,
		Root:       "sql",
	}
	_, err := migrate.Exec(connection, "postgres", migrations, direction)
	return err
}
