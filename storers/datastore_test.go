package storers

import (
	"context"
	"encoding/hex"
	"log"
	"os"
	"sync"

	"cloud.google.com/go/datastore"
	"google.golang.org/api/option"
	"impractical.co/auth/grants"

	"github.com/hashicorp/go-uuid"
)

func init() {
	if os.Getenv("DATASTORE_TEST_PROJECT") == "" || os.Getenv("DATASTORE_TEST_CREDS") == "" {
		return
	}
	client, err := datastore.NewClient(context.Background(), os.Getenv("DATASTORE_TEST_PROJECT"), option.WithServiceAccountFile(os.Getenv("DATASTORE_TEST_CREDS")))
	if err != nil {
		panic(err)
	}
	storerFactories = append(storerFactories, &DatastoreFactory{client: client})
}

type DatastoreFactory struct {
	client     *datastore.Client
	namespaces []string
	lock       sync.Mutex
}

func (d *DatastoreFactory) NewStorer(ctx context.Context) (grants.Storer, error) {
	namespaceSuffix, err := uuid.GenerateRandomBytes(6)
	if err != nil {
		log.Printf("Error generating UUID: %s", err.Error())
		return nil, err
	}
	namespace := "grants_test_" + hex.EncodeToString(namespaceSuffix)

	d.lock.Lock()
	d.namespaces = append(d.namespaces, namespace)
	d.lock.Unlock()

	storer, err := NewDatastore(ctx, d.client)
	if err != nil {
		return nil, err
	}
	storer.namespace = namespace

	return storer, nil
}

func (d *DatastoreFactory) TeardownStorers() error {
	d.lock.Lock()
	defer d.lock.Unlock()

	for _, namespace := range d.namespaces {
		var keys []*datastore.Key

		q := datastore.NewQuery(datastoreGrantKind).Namespace(namespace).KeysOnly()
		grantKeys, err := d.client.GetAll(context.Background(), q, nil)
		if err != nil {
			log.Printf("Error cleaning up grants in namespace %q: %s", namespace, err.Error())
			continue
		}
		keys = append(keys, grantKeys...)

		q = datastore.NewQuery(datastoreSourceRecordKind).Namespace(namespace).KeysOnly()
		sourceKeys, err := d.client.GetAll(context.Background(), q, nil)
		if err != nil {
			log.Printf("Error cleaning up source records in namespace %q: %s", namespace, err.Error())
			continue
		}
		keys = append(keys, sourceKeys...)

		err = d.client.DeleteMulti(context.Background(), keys)
		if err != nil {
			log.Printf("Error cleaning up grants and source records in namespace %q: %s", namespace, err.Error())
			continue
		}
	}
	return nil
}
