package grants

import (
	"sync"

	"golang.org/x/net/context"

	"github.com/pborman/uuid"
)

type Memstore struct {
	grants       map[uuid.Array]Grant
	grantSources map[string]struct{}
	lock         sync.Mutex
}

func NewMemstore() *Memstore {
	return &Memstore{
		grants:       map[uuid.Array]Grant{},
		grantSources: map[string]struct{}{},
	}
}

func (m *Memstore) CreateGrant(ctx context.Context, g Grant) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.grants[g.ID.Array()]; ok {
		return ErrGrantAlreadyExists
	}

	if _, ok := m.grantSources[g.SourceType+"."+g.SourceID]; ok {
		return ErrGrantSourceAlreadyUsed
	}

	m.grants[g.ID.Array()] = g
	m.grantSources[g.SourceType+"."+g.SourceID] = struct{}{}

	return nil
}

func (m *Memstore) ExchangeGrant(ctx context.Context, id uuid.UUID) (Grant, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	grant, ok := m.grants[id.Array()]
	if !ok {
		return Grant{}, ErrGrantNotFound
	}

	if grant.Used {
		return Grant{}, ErrGrantAlreadyUsed
	}
	grant.Used = true
	m.grants[id.Array()] = grant

	return grant, nil
}
