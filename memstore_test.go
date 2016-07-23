package grants

import "golang.org/x/net/context"

func init() {
	storerFactories = append(storerFactories, MemstoreFactory{})
}

type MemstoreFactory struct{}

func (m MemstoreFactory) NewStorer(ctx context.Context) (Storer, error) {
	return NewMemstore(), nil
}

func (m MemstoreFactory) TeardownStorers() error {
	return nil
}
