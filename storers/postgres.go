package storers

import (
	"context"
	"database/sql"

	"impractical.co/auth/grants"

	"github.com/lib/pq"

	"darlinggo.co/pan"
)

type Postgres struct {
	db *sql.DB
}

func NewPostgres(ctx context.Context, conn *sql.DB) (Postgres, error) {
	return Postgres{db: conn}, nil
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

func exchangeGrantUpdateSQL(id string) *pan.Query {
	var grant postgresGrant
	query := pan.New("UPDATE " + pan.Table(grant) + " SET ")
	query.Comparison(grant, "Used", "=", true)
	query.Where().Flush(" ")
	query.Comparison(grant, "ID", "=", id)
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

func (p Postgres) ExchangeGrant(ctx context.Context, id string) (grants.Grant, error) {
	query := exchangeGrantUpdateSQL(id)
	queryStr, err := query.PostgreSQLString()
	if err != nil {
		return grants.Grant{}, err
	}
	result, err := p.db.Exec(queryStr, query.Args()...)
	if err != nil {
		return grants.Grant{}, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return grants.Grant{}, err
	}
	query = exchangeGrantGetSQL(id)
	queryStr, err = query.PostgreSQLString()
	if err != nil {
		return grants.Grant{}, err
	}
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
	if grant.ID == "" {
		return fromPostgres(grant), grants.ErrGrantNotFound
	}
	if count < 1 {
		return grants.Grant{}, grants.ErrGrantAlreadyUsed
	}
	return fromPostgres(grant), nil
}
