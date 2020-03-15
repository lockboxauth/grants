package postgres

import (
	"context"
	"database/sql"

	"github.com/lib/pq"

	"darlinggo.co/pan"
	yall "yall.in"

	"lockbox.dev/grants"
)

//go:generate go-bindata -pkg migrations -o migrations/generated.go sql/

const (
	// TestConnStringEnvVar is the name of the environment variable
	// to set to the connection string when running tests.
	TestConnStringEnvVar = "PG_TEST_DB"
)

// Storer is a PostgreSQL implementation of the Storer
// interface.
type Storer struct {
	db *sql.DB
}

// NewStorer returns a PostgreSQL Storer instance that is ready
// to be used as a Storer.
func NewStorer(ctx context.Context, conn *sql.DB) Storer {
	return Storer{db: conn}
}

func createGrantSQL(grant Grant) *pan.Query {
	return pan.Insert(grant)
}

// CreateGrant inserts the passed Grant into the Storer,
// returning an ErrGrantAlreadyExists error if a Grant
// with the same ID alreday exists in the Storer, or am
// ErrGrantSourceAlreadyExists error if a Grant with the
// same SourceType and SourceID already exists in the Storer.
func (s Storer) CreateGrant(ctx context.Context, grant grants.Grant) error {
	query := createGrantSQL(toPostgres(grant))
	queryStr, err := query.PostgreSQLString()
	if err != nil {
		return err
	}
	_, err = s.db.Exec(queryStr, query.Args()...)
	if e, ok := err.(*pq.Error); ok {
		if e.Constraint == "grants_pkey" {
			err = grants.ErrGrantAlreadyExists
		} else if e.Constraint == "grants_source_type_source_id_key" {
			err = grants.ErrGrantSourceAlreadyUsed
		}
	}
	return err
}

func exchangeGrantUpdateSQL(g grants.GrantUse) *pan.Query {
	var grant Grant
	query := pan.New("UPDATE " + pan.Table(grant) + " SET ")
	query.Comparison(grant, "Used", "=", true)
	query.Comparison(grant, "UseIP", "=", g.IP)
	query.Comparison(grant, "UsedAt", "=", g.Time)
	query.Flush(", ").Where()
	query.Comparison(grant, "ID", "=", g.Grant)
	query.Comparison(grant, "Used", "=", false)
	return query.Flush(" AND ")
}

func exchangeGrantGetSQL(id string) *pan.Query {
	var grant Grant
	query := pan.New("SELECT " + pan.Columns(grant).String() + " FROM " + pan.Table(grant))
	query.Where()
	query.Comparison(grant, "ID", "=", id)
	return query.Flush(" ")
}

// ExchangeGrant applies the GrantUse to the Storer, marking
// the Grant in the Storer with an ID matching the Grant
// property of the GrantUse as used and recording metadata
// about the IP and time the Grant was used. If no Grant
// has an ID matching the Grant property of the GrantUse,
// an ErrGrantNotFound error is returned. If the Grant in
// the Storer with an ID matching the Grant propery of the
// GrantUse is already marked as used, an ErrGrantAlreadyUsed
// error will be returned.
func (s Storer) ExchangeGrant(ctx context.Context, g grants.GrantUse) (grants.Grant, error) {
	log := yall.FromContext(ctx).WithField("grant", g.Grant)
	// exchange the grant
	query := exchangeGrantUpdateSQL(g)
	queryStr, err := query.PostgreSQLString()
	if err != nil {
		return grants.Grant{}, err
	}
	log.WithField("query", queryStr).WithField("query_args", query.Args()).Debug("running update portion of grant exchange query")
	result, err := s.db.Exec(queryStr, query.Args()...)
	if err != nil {
		return grants.Grant{}, err
	}
	// figure out how many rows exchanging affected
	count, err := result.RowsAffected()
	if err != nil {
		return grants.Grant{}, err
	}
	log.WithField("rows_affected", count).Debug("successfully executed query")
	query = exchangeGrantGetSQL(g.Grant)
	queryStr, err = query.PostgreSQLString()
	if err != nil {
		return grants.Grant{}, err
	}
	log.WithField("query", queryStr).WithField("query_args", query.Args()).Debug("running get portion of grant exchange query")
	rows, err := s.db.Query(queryStr, query.Args()...)
	if err != nil {
		return grants.Grant{}, err
	}
	var grant Grant
	for rows.Next() {
		err = pan.Unmarshal(rows, &grant)
		if err != nil {
			return fromPostgres(grant), err
		}
	}
	if err = rows.Err(); err != nil {
		return fromPostgres(grant), err
	}
	// if we affected one or more rows, the exchange was
	// successfull, return the grant and we're done
	if count >= 1 {
		return fromPostgres(grant), nil
	}
	// if we affected fewer than one rows, the grant
	// wasn't successful.

	// if the grant doesn't exist in the Storer, that's an
	// ErrGrantNotFound error
	if grant.ID == "" {
		return fromPostgres(grant), grants.ErrGrantNotFound
	}
	// if the grant does exist in the Storer, the only reason
	// the exchange wouldn't have used it would be because it
	// has already been used. In that case, we want an
	// ErrGrantAlreadyUsed error.
	return grants.Grant{}, grants.ErrGrantAlreadyUsed
}

func getGrantSQL(id string) *pan.Query {
	var grant Grant
	query := pan.New("SELECT " + pan.Columns(grant).String() + " FROM " + pan.Table(grant))
	query.Where()
	query.Comparison(grant, "ID", "=", id)
	return query.Flush(" ")
}

// GetGrant retrieves the Grant specified by `id` from the Storer,
// returning an ErrGrantNotFound error if no Grant in the Storer
// has an ID matching `id`.
func (s Storer) GetGrant(ctx context.Context, id string) (grants.Grant, error) {
	log := yall.FromContext(ctx).WithField("grant", id)
	query := getGrantSQL(id)
	queryStr, err := query.PostgreSQLString()
	if err != nil {
		return grants.Grant{}, err
	}
	log.WithField("query", queryStr).WithField("query_args", query.Args()).Debug("running get grant query")
	rows, err := s.db.Query(queryStr, query.Args()...)
	if err != nil {
		return grants.Grant{}, err
	}
	var grant Grant
	for rows.Next() {
		err = pan.Unmarshal(rows, &grant)
		if err != nil {
			return fromPostgres(grant), err
		}
	}
	if err = rows.Err(); err != nil {
		return fromPostgres(grant), err
	}
	if grant.ID == "" {
		return fromPostgres(grant), grants.ErrGrantNotFound
	}
	return fromPostgres(grant), nil
}
