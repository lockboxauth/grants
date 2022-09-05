package memory

import (
	"context"
	"fmt"

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
func (s *Storer) CreateGrant(_ context.Context, grant grants.Grant) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	exists, err := txn.First("grant", "id", grant.ID)
	if err != nil {
		return err
	}
	if exists != nil {
		return grants.ErrGrantAlreadyExists
	}
	exists, err = txn.First("grant", "source", grant.SourceType, grant.SourceID)
	if err != nil {
		return err
	}
	if exists != nil {
		return grants.ErrGrantSourceAlreadyUsed
	}
	err = txn.Insert("grant", &grant)
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
func (s *Storer) ExchangeGrant(_ context.Context, use grants.GrantUse) (grants.Grant, error) {
	txn := s.db.Txn(true)
	defer txn.Abort()

	grant, err := txn.First("grant", "id", use.Grant)
	if err != nil {
		return grants.Grant{}, err
	}
	if grant == nil {
		return grants.Grant{}, grants.ErrGrantNotFound
	}

	found, ok := grant.(*grants.Grant)
	if !ok || found == nil {
		return grants.Grant{}, fmt.Errorf("unexpected result type %T", grant) //nolint:goerr113 // error for logging, not handling
	}
	newGrant := *found
	if newGrant.Used {
		return grants.Grant{}, grants.ErrGrantAlreadyUsed
	}
	if newGrant.Revoked {
		return grants.Grant{}, grants.ErrGrantRevoked
	}
	newGrant.Used = true
	newGrant.UseIP = use.IP
	newGrant.UsedAt = use.Time

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
func (s *Storer) GetGrant(_ context.Context, id string) (grants.Grant, error) {
	txn := s.db.Txn(false)
	grant, err := txn.First("grant", "id", id)
	if err != nil {
		return grants.Grant{}, err
	}
	if grant == nil {
		return grants.Grant{}, grants.ErrGrantNotFound
	}
	txn.Commit()

	res, ok := grant.(*grants.Grant)
	if !ok || res == nil {
		return grants.Grant{}, fmt.Errorf("unexpected result type %T", grant) //nolint:goerr113 // error for logging, not handling
	}

	return *res, nil
}

// GetGrantBySource retrieves the Grant specified by `sourceType` and
// `sourceID` from the Storer. If no Grant has a source type and source ID
// matching these parameters, an ErrGrantNotFound error is returned.
func (s *Storer) GetGrantBySource(_ context.Context, sourceType, sourceID string) (grants.Grant, error) {
	txn := s.db.Txn(false)
	grant, err := txn.First("grant", "source", sourceType, sourceID)
	if err != nil {
		return grants.Grant{}, err
	}
	if grant == nil {
		return grants.Grant{}, grants.ErrGrantNotFound
	}
	txn.Commit()

	res, ok := grant.(*grants.Grant)
	if !ok || res == nil {
		return grants.Grant{}, fmt.Errorf("unexpected result type %T", grant) //nolint:goerr113 // error for logging, not handling
	}

	return *res, nil
}

// RevokeGrant marks the Grant specified by `id` as revoked, meaning it can no
// longer be exchanged. If no Grant matches the specified ID, an
// ErrGrantNotFound error is returned. If the Grant matching the ID is already
// marked as revoked in the Storer, an ErrGrantRevoked error is returned. If
// the Grant matching the ID is already marked as used in the Storer, an
// ErrGrantAlreadyUsed error is returned.
func (s *Storer) RevokeGrant(_ context.Context, id string) (grants.Grant, error) {
	txn := s.db.Txn(true)
	defer txn.Abort()

	grant, err := txn.First("grant", "id", id)
	if err != nil {
		return grants.Grant{}, err
	}
	if grant == nil {
		return grants.Grant{}, grants.ErrGrantNotFound
	}

	found, ok := grant.(*grants.Grant)
	if !ok || found == nil {
		return grants.Grant{}, fmt.Errorf("unexpected result type %T", grant) //nolint:goerr113 // error for logging, not handling
	}
	newGrant := *found
	if newGrant.Used {
		return grants.Grant{}, grants.ErrGrantAlreadyUsed
	}
	if newGrant.Revoked {
		return grants.Grant{}, grants.ErrGrantRevoked
	}
	newGrant.Revoked = true

	err = txn.Insert("grant", &newGrant)
	if err != nil {
		return grants.Grant{}, err
	}
	txn.Commit()

	return newGrant, nil
}
