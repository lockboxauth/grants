package postgres

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/url"
	"os"
	"sync"

	"lockbox.dev/grants"

	uuid "github.com/hashicorp/go-uuid"
	migrate "github.com/rubenv/sql-migrate"
)

// Factory implements the grants.Factory interface
// for the Storer type; it offers a consistent
// interface for setting up and tearing down Storers
// for testing purposes.
type Factory struct {
	db        *sql.DB
	databases map[string]*sql.DB
	lock      sync.Mutex
}

// NewFactory returns a Factory, ready to be used.
// NewFactory must be called to obtain a usable Factory,
// because Factory types have internal state that must
// be initialized.
func NewFactory(db *sql.DB) *Factory {
	return &Factory{
		db:        db,
		databases: map[string]*sql.DB{},
	}
}

// NewStorer creates a new Storer and returns it.
func (f *Factory) NewStorer(ctx context.Context) (grants.Storer, error) { //nolint:ireturn // interface requires returning an interface
	connString, err := url.Parse(os.Getenv(TestConnStringEnvVar))
	if err != nil {
		log.Printf("Error parsing %s as a URL: %+v\n", TestConnStringEnvVar, err)
		return nil, err
	}
	if connString.Scheme != "postgres" {
		return nil, fmt.Errorf("%s must begin with postgres://", TestConnStringEnvVar) //nolint:goerr113 // error isn't handled, only for display
	}

	tableSuffix, err := uuid.GenerateRandomBytes(6) //nolint:gomnd // number is arbitrary, not magic
	if err != nil {
		log.Printf("Error generating UUID: %+v\n", err)
		return nil, err
	}
	table := "grants_test_" + hex.EncodeToString(tableSuffix)

	_, err = f.db.Exec("CREATE DATABASE " + table + ";")
	if err != nil {
		log.Printf("Error creating database %s: %+v\n", table, err)
		return nil, err
	}

	connString.Path = "/" + table
	newConn, err := sql.Open("postgres", connString.String())
	if err != nil {
		log.Println("Accidentally orphaned", table, "it will need to be cleaned up manually")
		return nil, err
	}

	f.lock.Lock()
	if f.databases == nil {
		f.databases = map[string]*sql.DB{}
	}
	f.databases[table] = newConn
	f.lock.Unlock()

	err = ApplyMigrations(newConn, migrate.Up)
	if err != nil {
		return nil, err
	}

	storer := NewStorer(ctx, newConn)
	return storer, nil
}

// TeardownStorers drops all the databases created by NewStorer,
// cleaning up after the Factory.
func (f *Factory) TeardownStorers() error {
	f.lock.Lock()
	defer f.lock.Unlock()
	for table, conn := range f.databases {
		err := conn.Close()
		if err != nil {
			return err
		}
		_, err = f.db.Exec("DROP DATABASE " + table + ";")
		if err != nil {
			return err
		}
	}
	return f.db.Close()
}
