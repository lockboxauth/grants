package storers

import (
	"context"

	"code.impractical.co/grants"
)

func init() {
	storerFactories = append(storerFactories, MemstoreFactory{})
}

type MemstoreFactory struct{}

func (m MemstoreFactory) NewStorer(ctx context.Context) (grants.Storer, error) {
	return NewMemstore(), nil
}

func (m MemstoreFactory) TeardownStorers() error {
	return nil
}
