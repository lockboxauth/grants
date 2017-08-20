package storers

import (
	"context"

	"impractical.co/auth/grants"
)

func init() {
	storerFactories = append(storerFactories, MemstoreFactory{})
}

type MemstoreFactory struct{}

func (m MemstoreFactory) NewStorer(ctx context.Context) (grants.Storer, error) {
	return NewMemstore()
}

func (m MemstoreFactory) TeardownStorers() error {
	return nil
}
