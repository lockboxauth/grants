package memory

import (
	"context"

	"lockbox.dev/grants"
)

// Factory implements the grants.Factory interface
// for the Storer type; it offers a consistent
// interface for setting up and tearing down Storers
// for testing purposes.
type Factory struct{}

// NewStorer creates a new Storer and returns it.
func (f Factory) NewStorer(ctx context.Context) (grants.Storer, error) {
	return NewStorer()
}

// TeardownStorers does nothing, as Storers need no
// teardown. It exists to fill the Factory interface.
func (f Factory) TeardownStorers() error {
	return nil
}
