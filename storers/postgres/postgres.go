package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"darlinggo.co/pan"
	yall "yall.in"

	"lockbox.dev/grants"
)

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
func NewStorer(_ context.Context, conn *sql.DB) Storer {
	return Storer{db: conn}
}

func createGrantSQL(grant Grant) *pan.Query {
	return pan.Insert(grant)
}

func createGrantAncestorsSQL(ancestors []GrantAncestor) *pan.Query {
	namer := make([]pan.SQLTableNamer, 0, len(ancestors))
	for _, anc := range ancestors {
		namer = append(namer, anc)
	}
	return pan.Insert(namer...)
}

// CreateGrant inserts the passed Grant into the Storer,
// returning an ErrGrantAlreadyExists error if a Grant
// with the same ID alreday exists in the Storer, or am
// ErrGrantSourceAlreadyExists error if a Grant with the
// same SourceType and SourceID already exists in the Storer.
func (s Storer) CreateGrant(ctx context.Context, grant grants.Grant) error {
	grantQuery := createGrantSQL(toPostgres(grant))
	grantQueryStr, err := grantQuery.PostgreSQLString()
	if err != nil {
		return err
	}
	var ancestorQuery *pan.Query
	var ancestorQueryStr string
	if len(grant.AncestorIDs) > 0 {
		ancestorQuery = createGrantAncestorsSQL(ancestorsFromIDs(grant.ID, grant.AncestorIDs))
		ancestorQueryStr, err = ancestorQuery.PostgreSQLString()
		if err != nil {
			return err
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, grantQueryStr, grantQuery.Args()...)
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Constraint {
		case "grants_pkey":
			err = grants.ErrGrantAlreadyExists
		case "grants_source_type_source_id_key":
			err = grants.ErrGrantSourceAlreadyUsed
		}
	}
	if err != nil {
		tx.Rollback()
		return err
	}

	if ancestorQuery != nil && ancestorQueryStr != "" {
		_, err = tx.ExecContext(ctx, ancestorQueryStr, ancestorQuery.Args()...)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	err = tx.Commit()
	return err
}

func exchangeGrantUpdateSQL(use grants.GrantUse) *pan.Query {
	var grant Grant
	query := pan.New("UPDATE " + pan.Table(grant) + " SET ")
	query.Comparison(grant, "Used", "=", true)
	query.Comparison(grant, "UseIP", "=", use.IP)
	query.Comparison(grant, "UsedAt", "=", use.Time)
	query.Flush(", ").Where()
	query.Comparison(grant, "ID", "=", use.Grant)
	query.Comparison(grant, "Used", "=", false)
	query.Comparison(grant, "Revoked", "=", false)
	return query.Flush(" AND ")
}

func exchangeGrantGetSQL(id string) *pan.Query {
	var grant Grant
	query := pan.New("SELECT " + pan.Columns(grant).String() + " FROM " + pan.Table(grant))
	query.Where()
	query.Comparison(grant, "ID", "=", id)
	return query.Flush(" ")
}

func getAncestorsSQL(id string) *pan.Query {
	var ancestor GrantAncestor
	query := pan.New("SELECT " + pan.Columns(ancestor).String() + " FROM " + pan.Table(ancestor))
	query.Where()
	query.Comparison(ancestor, "GrantID", "=", id)
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
func (s Storer) ExchangeGrant(ctx context.Context, use grants.GrantUse) (grants.Grant, error) {
	log := yall.FromContext(ctx).WithField("grant", use.Grant)
	// exchange the grant
	query := exchangeGrantUpdateSQL(use)
	queryStr, err := query.PostgreSQLString()
	if err != nil {
		return grants.Grant{}, err
	}
	log.WithField("query", queryStr).WithField("query_args", query.Args()).Debug("running update portion of grant exchange query")
	result, err := s.db.ExecContext(ctx, queryStr, query.Args()...)
	if err != nil {
		return grants.Grant{}, err
	}
	// figure out how many rows exchanging affected
	count, err := result.RowsAffected()
	if err != nil {
		return grants.Grant{}, err
	}
	log.WithField("rows_affected", count).Debug("successfully executed query")
	query = exchangeGrantGetSQL(use.Grant)
	queryStr, err = query.PostgreSQLString()
	if err != nil {
		return grants.Grant{}, err
	}
	log.WithField("query", queryStr).WithField("query_args", query.Args()).Debug("running get portion of grant exchange query")
	rows, err := s.db.QueryContext(ctx, queryStr, query.Args()...) //nolint:sqlclosecheck // the closeRows helper isn't picked up
	if err != nil {
		return grants.Grant{}, err
	}
	defer closeRows(ctx, rows)
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
	query = getAncestorsSQL(use.Grant)
	queryStr, err = query.PostgreSQLString()
	if err != nil {
		return grants.Grant{}, err
	}
	log.WithField("query", queryStr).WithField("query_args", query.Args()).Debug("running get ancestors portion of grant exchange query")
	ancestorRows, err := s.db.QueryContext(ctx, queryStr, query.Args()...) //nolint:sqlclosecheck // the closeRows helper isn't picked up
	if err != nil {
		return grants.Grant{}, err
	}
	defer closeRows(ctx, ancestorRows)
	for ancestorRows.Next() {
		var ancestor GrantAncestor
		err = pan.Unmarshal(ancestorRows, &ancestor)
		if err != nil {
			return fromPostgres(grant), err
		}
		grant.Ancestors = append(grant.Ancestors, ancestor)
	}
	if err = ancestorRows.Err(); err != nil {
		return fromPostgres(grant), err
	}
	// if we affected one or more rows, the exchange was
	// successful, return the grant and we're done
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
	// if the Grant exists but we didn't update it, either it was already
	// used or was revoked.
	if grant.Used {
		return grants.Grant{}, grants.ErrGrantAlreadyUsed
	}
	if grant.Revoked {
		return grants.Grant{}, grants.ErrGrantRevoked
	}
	return grants.Grant{}, fmt.Errorf("error exchanging %s: %w", use.Grant, errors.New("unexpected error, no grants updated, grant found, grant not used or revoked"))
}

func revokeGrantUpdateSQL(id string) *pan.Query {
	var grant Grant
	query := pan.New("UPDATE " + pan.Table(grant) + " SET ")
	query.Comparison(grant, "Revoked", "=", true)
	query.Flush(", ").Where()
	query.Comparison(grant, "ID", "=", id)
	query.Comparison(grant, "Used", "=", false)
	query.Comparison(grant, "Revoked", "=", false)
	return query.Flush(" AND ")
}

func revokeGrantGetSQL(id string) *pan.Query {
	var grant Grant
	query := pan.New("SELECT " + pan.Columns(grant).String() + " FROM " + pan.Table(grant))
	query.Where()
	query.Comparison(grant, "ID", "=", id)
	return query.Flush(" ")
}

// RevokeGrant marks the Grant specified by id as revoked in the Storer, making
// in unable to be exchanged. If no Grant has an ID matching the passed id, an
// ErrGrantNotFound error is returned. If the Grant in the Storer with an ID
// matching the passed id is already marked as used, an ErrGrantAlreadyUsed
// error will be returned. If the Grant in the Storer with an ID matching the
// passed id is already marked as revoked, an ErrGrantRevoked error will be
// returned.
func (s Storer) RevokeGrant(ctx context.Context, id string) (grants.Grant, error) {
	log := yall.FromContext(ctx).WithField("grant", id)
	// revoke the grant
	query := revokeGrantUpdateSQL(id)
	queryStr, err := query.PostgreSQLString()
	if err != nil {
		return grants.Grant{}, err
	}
	log.WithField("query", queryStr).WithField("query_args", query.Args()).Debug("running update portion of grant revoke query")
	result, err := s.db.ExecContext(ctx, queryStr, query.Args()...)
	if err != nil {
		return grants.Grant{}, err
	}
	// figure out how many rows exchanging affected
	count, err := result.RowsAffected()
	if err != nil {
		return grants.Grant{}, err
	}
	log.WithField("rows_affected", count).Debug("successfully executed query")
	query = revokeGrantGetSQL(id)
	queryStr, err = query.PostgreSQLString()
	if err != nil {
		return grants.Grant{}, err
	}
	log.WithField("query", queryStr).WithField("query_args", query.Args()).Debug("running get portion of grant revoke query")
	rows, err := s.db.QueryContext(ctx, queryStr, query.Args()...) //nolint:sqlclosecheck // the closeRows helper isn't picked up
	if err != nil {
		return grants.Grant{}, err
	}
	defer closeRows(ctx, rows)
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
	query = getAncestorsSQL(id)
	queryStr, err = query.PostgreSQLString()
	if err != nil {
		return grants.Grant{}, err
	}
	log.WithField("query", queryStr).WithField("query_args", query.Args()).Debug("running get ancestors portion of grant revoke query")
	ancestorRows, err := s.db.QueryContext(ctx, queryStr, query.Args()...) //nolint:sqlclosecheck // the closeRows helper isn't picked up
	if err != nil {
		return grants.Grant{}, err
	}
	defer closeRows(ctx, ancestorRows)
	for ancestorRows.Next() {
		var ancestor GrantAncestor
		err = pan.Unmarshal(ancestorRows, &ancestor)
		if err != nil {
			return fromPostgres(grant), err
		}
		grant.Ancestors = append(grant.Ancestors, ancestor)
	}
	if err = ancestorRows.Err(); err != nil {
		return fromPostgres(grant), err
	}
	// if we affected one or more rows, the revoke was
	// successful, return the grant and we're done
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
	// if the Grant exists but we didn't update it, either it was already
	// used or was revoked.
	if grant.Used {
		return grants.Grant{}, grants.ErrGrantAlreadyUsed
	}
	if grant.Revoked {
		return grants.Grant{}, grants.ErrGrantRevoked
	}
	return grants.Grant{}, fmt.Errorf("error revoking %s: %w", id, errors.New("unexpected error, no grants updated, grant found, grant not used or revoked"))
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
	rows, err := s.db.QueryContext(ctx, queryStr, query.Args()...) //nolint:sqlclosecheck // the closeRows helper isn't picked up
	if err != nil {
		return grants.Grant{}, err
	}
	defer closeRows(ctx, rows)
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
	query = getAncestorsSQL(id)
	queryStr, err = query.PostgreSQLString()
	if err != nil {
		return grants.Grant{}, err
	}
	log.WithField("query", queryStr).WithField("query_args", query.Args()).Debug("running get ancestors portion of get grant query")
	ancestorRows, err := s.db.QueryContext(ctx, queryStr, query.Args()...) //nolint:sqlclosecheck // the closeRows helper isn't picked up
	if err != nil {
		return grants.Grant{}, err
	}
	defer closeRows(ctx, ancestorRows)
	for ancestorRows.Next() {
		var ancestor GrantAncestor
		err = pan.Unmarshal(ancestorRows, &ancestor)
		if err != nil {
			return fromPostgres(grant), err
		}
		grant.Ancestors = append(grant.Ancestors, ancestor)
	}
	if err = ancestorRows.Err(); err != nil {
		return fromPostgres(grant), err
	}
	return fromPostgres(grant), nil
}

func getGrantBySourceSQL(sourceType, sourceID string) *pan.Query {
	var grant Grant
	query := pan.New("SELECT " + pan.Columns(grant).String() + " FROM " + pan.Table(grant))
	query.Where()
	query.Comparison(grant, "SourceType", "=", sourceType)
	query.Comparison(grant, "SourceID", "=", sourceID)
	return query.Flush(" AND ")
}

// GetGrantBySource retrieves the Grant specified by `sourceType` and
// `sourceID` from the Storer, returning an ErrGrantNotFound error if no Grant
// in the Storer has a SourceType and SourceID matching those parameters.
func (s Storer) GetGrantBySource(ctx context.Context, sourceType, sourceID string) (grants.Grant, error) {
	log := yall.FromContext(ctx).WithField("source_type", sourceType)
	log = log.WithField("source_id", sourceID)
	query := getGrantBySourceSQL(sourceType, sourceID)
	queryStr, err := query.PostgreSQLString()
	if err != nil {
		return grants.Grant{}, err
	}
	log.WithField("query", queryStr).WithField("query_args", query.Args()).Debug("running get grant by source query")
	rows, err := s.db.QueryContext(ctx, queryStr, query.Args()...) //nolint:sqlclosecheck // the closeRows helper isn't picked up
	if err != nil {
		return grants.Grant{}, err
	}
	defer closeRows(ctx, rows)
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
	query = getAncestorsSQL(grant.ID)
	queryStr, err = query.PostgreSQLString()
	if err != nil {
		return grants.Grant{}, err
	}
	log.WithField("query", queryStr).WithField("query_args", query.Args()).Debug("running ancestors portion of get grant by source query")
	ancestorRows, err := s.db.QueryContext(ctx, queryStr, query.Args()...) //nolint:sqlclosecheck // the closeRows helper isn't picked up
	if err != nil {
		return grants.Grant{}, err
	}
	defer closeRows(ctx, ancestorRows)
	for ancestorRows.Next() {
		var ancestor GrantAncestor
		err = pan.Unmarshal(ancestorRows, &ancestor)
		if err != nil {
			return fromPostgres(grant), err
		}
		grant.Ancestors = append(grant.Ancestors, ancestor)
	}
	if err = ancestorRows.Err(); err != nil {
		return fromPostgres(grant), err
	}
	return fromPostgres(grant), nil
}

func closeRows(ctx context.Context, rows *sql.Rows) {
	if err := rows.Close(); err != nil {
		yall.FromContext(ctx).WithError(err).Error("failed to close rows")
	}
}
