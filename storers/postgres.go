package storers

import (
	"context"
	"database/sql"

	"impractical.co/auth/grants"
	yall "yall.in"

	"github.com/lib/pq"

	"darlinggo.co/pan"
)

type Postgres struct {
	db *sql.DB
}

func NewPostgres(ctx context.Context, conn *sql.DB) Postgres {
	return Postgres{db: conn}
}

func createGrantSQL(grant postgresGrant) *pan.Query {
	return pan.Insert(grant)
}

func (p Postgres) CreateGrant(ctx context.Context, grant grants.Grant) error {
	query := createGrantSQL(toPostgres(grant))
	queryStr, err := query.PostgreSQLString()
	if err != nil {
		return err
	}
	_, err = p.db.Exec(queryStr, query.Args()...)
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
	var grant postgresGrant
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
	var grant postgresGrant
	query := pan.New("SELECT " + pan.Columns(grant).String() + " FROM " + pan.Table(grant))
	query.Where()
	query.Comparison(grant, "ID", "=", id)
	return query.Flush(" ")
}

func (p Postgres) ExchangeGrant(ctx context.Context, g grants.GrantUse) (grants.Grant, error) {
	log := yall.FromContext(ctx).WithField("grant", g.Grant)
	// exchange the grant
	query := exchangeGrantUpdateSQL(g)
	queryStr, err := query.PostgreSQLString()
	if err != nil {
		return grants.Grant{}, err
	}
	log.WithField("query", queryStr).WithField("query_args", query.Args()).Debug("running update portion of grant exchange query")
	result, err := p.db.Exec(queryStr, query.Args()...)
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
	rows, err := p.db.Query(queryStr, query.Args()...)
	if err != nil {
		return grants.Grant{}, err
	}
	var grant postgresGrant
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
