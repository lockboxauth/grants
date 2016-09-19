package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"

	"code.impractical.co/grants"

	"github.com/rubenv/sql-migrate"

	"darlinggo.co/healthcheck"
	"darlinggo.co/version"
)

func main() {
	// Set up postgres connection
	postgres := os.Getenv("PG_DB")
	if postgres == "" {
		log.Println("Error setting up Postgres: no connection string set.")
		os.Exit(1)
	}
	db, err := sql.Open("postgres", postgres)
	if err != nil {
		log.Printf("Error connecting to Postgres: %+v\n", err)
		os.Exit(1)
	}
	migrations := &migrate.AssetMigrationSource{
		Asset:    grants.Asset,
		AssetDir: grants.AssetDir,
		Dir:      "sql",
	}
	_, err = migrate.Exec(db, "postgres", migrations, migrate.Up)
	if err != nil {
		log.Printf("Error running migrations for Postgres: %+v\n", err)
		os.Exit(1)
	}
	/*storer, err := grants.NewPostgres(ctx, db)
	if err != nil {
		log.Printf("Error setting up Postgres: %+v\n", err)
		os.Exit(1)
	}
	v1 := apiv1.APIv1{Dependencies: grants.Dependencies{Storer: storer}}

	// we need both to avoid redirecting, which turns POST into GET
	// the slash is needed to handle /v1/*
	http.Handle("/v1/", v1.Server(ctx, "/v1/"))
	http.Handle("/v1", v1.Server(ctx, "/v1"))
	*/

	// set up version handler
	http.Handle("/version", version.Handler)

	// set up health check
	dbCheck := healthcheck.NewSQL(db, "main Postgres DB")
	checker := healthcheck.NewChecks(context.Background(), log.Printf, dbCheck)
	http.Handle("/health", checker)

	vers := version.Tag
	if vers == "undefined" || vers == "" {
		vers = "dev"
	}
	vers = vers + " (" + version.Hash + ")"

	log.Printf("grantsd version %s starting on port 0.0.0.0:4002\n", vers)
	err = http.ListenAndServe("0.0.0.0:4002", nil)
	if err != nil {
		log.Printf("Error listening on port 0.0.0.0:4002: %+v\n", err)
		os.Exit(1)
	}
}
