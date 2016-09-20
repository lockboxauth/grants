package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"

	"code.impractical.co/grants"
	"code.impractical.co/grants/apiv1"
	refresh "code.impractical.co/tokens/client"

	"github.com/rubenv/sql-migrate"

	"darlinggo.co/healthcheck"
	"darlinggo.co/version"
)

func main() {
	// Set up our logger
	logger := log.New(os.Stdout, "", log.Llongfile|log.LstdFlags|log.LUTC|log.Lmicroseconds)

	// Set up postgres connection
	postgres := os.Getenv("PG_DB")
	if postgres == "" {
		logger.Println("Error setting up Postgres: no connection string set.")
		os.Exit(1)
	}
	db, err := sql.Open("postgres", postgres)
	if err != nil {
		logger.Printf("Error connecting to Postgres: %+v\n", err)
		os.Exit(1)
	}

	// set up our tokensd client
	tokensdBaseURL := os.Getenv("TOKENSD_BASEURL")
	if tokensdBaseURL == "" {
		logger.Println("Error setting up tokensd client: no base URL set.")
		os.Exit(1)
	}

	// set up our base context
	ctx := context.Background()

	// run our postgres migrations
	migrations := &migrate.AssetMigrationSource{
		Asset:    grants.Asset,
		AssetDir: grants.AssetDir,
		Dir:      "sql",
	}
	_, err = migrate.Exec(db, "postgres", migrations, migrate.Up)
	if err != nil {
		logger.Printf("Error running migrations for Postgres: %+v\n", err)
		os.Exit(1)
	}

	// build our APIv1 struct
	storer, err := grants.NewPostgres(ctx, db)
	if err != nil {
		logger.Printf("Error setting up Postgres: %+v\n", err)
		os.Exit(1)
	}
	refreshTokenClient := refresh.NewAPIManager(http.DefaultClient, tokensdBaseURL, "grantsd")
	v1 := apiv1.APIv1{
		Dependencies: grants.Dependencies{
			Storer: storer,
			Log:    logger,
		},
		Tokens: refreshTokenClient,
	}

	// set up our APIv1 handlers
	// we need both to avoid redirecting, which turns POST into GET
	// the slash is needed to handle /v1/
	http.Handle("/v1/", v1.Server("/v1/"))
	http.Handle("/v1", v1.Server("/v1"))

	// set up version handler
	http.Handle("/version", version.Handler)

	// set up health check
	dbCheck := healthcheck.NewSQL(db, "main Postgres DB")
	checker := healthcheck.NewChecks(ctx, logger.Printf, dbCheck)
	http.Handle("/health", checker)

	// make our version information pretty
	vers := version.Tag
	if vers == "undefined" || vers == "" {
		vers = "dev"
	}
	vers = vers + " (" + version.Hash + ")"

	// let users know what's listening and on what address/port
	logger.Printf("grantsd version %s starting on port 0.0.0.0:4002\n", vers)
	err = http.ListenAndServe("0.0.0.0:4002", nil)
	if err != nil {
		logger.Printf("Error listening on port 0.0.0.0:4002: %+v\n", err)
		os.Exit(1)
	}
}
