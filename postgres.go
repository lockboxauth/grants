package grants

import (
	"database/sql"

	"github.com/lib/pq"

	"github.com/pborman/uuid"

	"darlinggo.co/pan"
	"golang.org/x/net/context"
)

type Postgres struct {
	db *sql.DB
}

func NewPostgres(ctx context.Context, conn *sql.DB) (Postgres, error) {
	return Postgres{db: conn}, nil
}

func (g Grant) GetSQLTableName() string {
	return "grants"
}

func createGrantSQL(grant Grant) *pan.Query {
	fields, values := pan.GetFields(grant)
	query := pan.New(pan.POSTGRES, "INSERT INTO "+pan.GetTableName(grant))
	query.Include("(" + pan.QueryList(fields) + ")")
	query.Include("VALUES")
	query.Include("("+pan.VariableList(len(values))+")", values...)
	return query.FlushExpressions(" ")
}

func (p Postgres) CreateGrant(ctx context.Context, grant Grant) error {
	query := createGrantSQL(grant)
	_, err := p.db.Exec(query.String(), query.Args...)
	if e, ok := err.(*pq.Error); ok {
		if e.Constraint == "grants_pkey" {
			err = ErrGrantAlreadyExists
		} else if e.Constraint == "grants_source_type_source_id_key" {
			err = ErrGrantSourceAlreadyUsed
		}
	}
	return err
}

func exchangeGrantUpdateSQL(id uuid.UUID) *pan.Query {
	var grant Grant
	query := pan.New(pan.POSTGRES, "UPDATE "+pan.GetTableName(grant)+" SET ")
	query.Include(pan.GetUnquotedColumn(grant, "Used")+" = ?", true)
	query.IncludeWhere()
	query.FlushExpressions(" ")
	query.Include(pan.GetUnquotedColumn(grant, "ID")+" = ?", id)
	query.Include(pan.GetUnquotedColumn(grant, "Used")+" = ?", false)
	return query.FlushExpressions(" AND ")
}

func exchangeGrantGetSQL(id uuid.UUID) *pan.Query {
	var grant Grant
	fields, _ := pan.GetFields(grant)
	query := pan.New(pan.POSTGRES, "SELECT "+pan.QueryList(fields)+" FROM "+pan.GetTableName(grant))
	query.IncludeWhere()
	query.Include(pan.GetUnquotedColumn(grant, "ID")+" = ?", id)
	return query.FlushExpressions(" ")
}

func (p Postgres) ExchangeGrant(ctx context.Context, id uuid.UUID) (Grant, error) {
	query := exchangeGrantUpdateSQL(id)
	result, err := p.db.Exec(query.String(), query.Args...)
	if err != nil {
		return Grant{}, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return Grant{}, err
	}
	query = exchangeGrantGetSQL(id)
	rows, err := p.db.Query(query.String(), query.Args...)
	if err != nil {
		return Grant{}, err
	}
	var grant Grant
	for rows.Next() {
		err = pan.Unmarshal(rows, &grant)
		if err != nil {
			return grant, err
		}
	}
	if err = rows.Err(); err != nil {
		return grant, err
	}
	if grant.ID == nil {
		return grant, ErrGrantNotFound
	}
	if count < 1 {
		return Grant{}, ErrGrantAlreadyUsed
	}
	return grant, nil
}
