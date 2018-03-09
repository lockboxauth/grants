package storers

import (
	"context"

	"cloud.google.com/go/datastore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"impractical.co/auth/grants"
)

const (
	datastoreGrantKind        = "Grant"
	datastoreSourceRecordKind = "SourceRecord"
)

type Datastore struct {
	client    *datastore.Client
	namespace string
}

func NewDatastore(ctx context.Context, client *datastore.Client) (*Datastore, error) {
	return &Datastore{client: client}, nil
}

func (d *Datastore) key(id string) *datastore.Key {
	key := datastore.NameKey(datastoreGrantKind, id, nil)
	if d.namespace != "" {
		key.Namespace = d.namespace
	}
	return key
}

func (d *Datastore) sourceKey(sourceType, sourceID string) *datastore.Key {
	key := datastore.NameKey(datastoreSourceRecordKind, sourceType+"::"+sourceID, nil)
	if d.namespace != "" {
		key.Namespace = d.namespace
	}
	return key
}

func (d *Datastore) CreateGrant(ctx context.Context, grant grants.Grant) error {
	gr := toDatastore(grant)
	mut := datastore.NewInsert(d.sourceKey(gr.SourceType, gr.SourceID), &datastoreSourceRecord{
		ID:         gr.ID,
		SourceType: gr.SourceType,
		SourceID:   gr.SourceID,
		CreatedAt:  gr.CreatedAt,
	})
	_, err := d.client.Mutate(ctx, mut)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return grants.ErrGrantSourceAlreadyUsed
		}
		return err
	}
	mut = datastore.NewInsert(d.key(gr.ID), &gr)
	_, err = d.client.Mutate(ctx, mut)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return grants.ErrGrantAlreadyExists
		}
		return err
	}
	return nil
}

func (d *Datastore) ExchangeGrant(ctx context.Context, id string) (grants.Grant, error) {
	var gr grants.Grant
	_, err := d.client.RunInTransaction(ctx, func(txn *datastore.Transaction) error {
		var grant datastoreGrant
		err := txn.Get(d.key(id), &grant)
		if err == datastore.ErrNoSuchEntity {
			return grants.ErrGrantNotFound
		} else if err != nil {
			return err
		}
		if grant.Used {
			return grants.ErrGrantAlreadyUsed
		}
		grant.Used = true
		_, err = txn.Put(d.key(id), &grant)
		if err != nil {
			return err
		}
		gr = fromDatastore(grant)
		return nil
	})
	if err != nil {
		return gr, err
	}
	return gr, nil
}
