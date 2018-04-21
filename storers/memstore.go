package storers

import (
	"context"

	"impractical.co/auth/grants"

	memdb "github.com/hashicorp/go-memdb"
)

var (
	schema = &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"grant": &memdb.TableSchema{
				Name: "grant",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:   "id",
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "ID",
						},
					},
					"source": &memdb.IndexSchema{
						Name:   "source",
						Unique: true,
						Indexer: &memdb.CompoundIndex{
							Indexes: []memdb.Indexer{
								&memdb.StringFieldIndex{
									Field:     "SourceType",
									Lowercase: true,
								},
								&memdb.StringFieldIndex{
									Field:     "SourceID",
									Lowercase: true,
								},
							},
						},
					},
				},
			},
		},
	}
)

type Memstore struct {
	db *memdb.MemDB
}

func NewMemstore() (*Memstore, error) {
	db, err := memdb.NewMemDB(schema)
	if err != nil {
		return nil, err
	}
	return &Memstore{
		db: db,
	}, nil
}

func (m *Memstore) CreateGrant(ctx context.Context, g grants.Grant) error {
	txn := m.db.Txn(true)
	defer txn.Abort()

	exists, err := txn.First("grant", "id", g.ID)
	if err != nil {
		return err
	}
	if exists != nil {
		return grants.ErrGrantAlreadyExists
	}
	exists, err = txn.First("grant", "source", g.SourceType, g.SourceID)
	if err != nil {
		return err
	}
	if exists != nil {
		return grants.ErrGrantSourceAlreadyUsed
	}
	err = txn.Insert("grant", &g)
	if err != nil {
		return err
	}
	txn.Commit()
	return nil
}

func (m *Memstore) ExchangeGrant(ctx context.Context, g grants.GrantUse) (grants.Grant, error) {
	txn := m.db.Txn(true)
	defer txn.Abort()

	grant, err := txn.First("grant", "id", g.Grant)
	if err != nil {
		return grants.Grant{}, err
	}
	if grant == nil {
		return grants.Grant{}, grants.ErrGrantNotFound
	}

	if grant.(*grants.Grant).Used {
		return grants.Grant{}, grants.ErrGrantAlreadyUsed
	}
	newGrant := *grant.(*grants.Grant)
	newGrant.Used = true
	newGrant.UseIP = g.IP
	newGrant.UsedAt = g.Time

	err = txn.Insert("grant", &newGrant)
	if err != nil {
		return grants.Grant{}, err
	}
	txn.Commit()

	return newGrant, nil
}
