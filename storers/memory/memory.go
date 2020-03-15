package memory

import (
	"context"

	memdb "github.com/hashicorp/go-memdb"

	"lockbox.dev/grants"
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

// Storer is an in-memory implementation of the Storer
// interface.
type Storer struct {
	db *memdb.MemDB
}

// NewStorer returns an in-memory Storer instance that is ready
// to be used as a Storer.
func NewStorer() (*Storer, error) {
	db, err := memdb.NewMemDB(schema)
	if err != nil {
		return nil, err
	}
	return &Storer{
		db: db,
	}, nil
}

// CreateGrant inserts the passed Grant into the Storer,
// returning an ErrGrantAlreadyExists error if a Grant
// with the same ID alreday exists in the Storer, or am
// ErrGrantSourceAlreadyExists error if a Grant with the
// same SourceType and SourceID already exists in the Storer.
func (s *Storer) CreateGrant(ctx context.Context, g grants.Grant) error {
	txn := s.db.Txn(true)
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

// ExchangeGrant applies the GrantUse to the Storer, marking
// the Grant in the Storer with an ID matching the Grant
// property of the GrantUse as used and recording metadata
// about the IP and time the Grant was used. If no Grant
// has an ID matching the Grant property of the GrantUse,
// an ErrGrantNotFound error is returned. If the Grant in
// the Storer with an ID matching the Grant propery of the
// GrantUse is already marked as used, an ErrGrantAlreadyUsed
// error will be returned.
func (s *Storer) ExchangeGrant(ctx context.Context, g grants.GrantUse) (grants.Grant, error) {
	txn := s.db.Txn(true)
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

// GetGrant retrieves the Grant specified by `id` from the
// Storer. If no Grant has an ID matching the `id` parameter,
// an ErrGrantNotFound error is returned.
func (s *Storer) GetGrant(ctx context.Context, id string) (grants.Grant, error) {
	txn := s.db.Txn(false)
	grant, err := txn.First("grant", "id", id)
	if err != nil {
		return grants.Grant{}, err
	}
	if grant == nil {
		return grants.Grant{}, grants.ErrGrantNotFound
	}
	txn.Commit()

	return *grant.(*grants.Grant), nil
}
