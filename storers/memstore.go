package storers

import (
	"context"
	"sync"

	"code.impractical.co/grants"

	"github.com/pborman/uuid"
)

type Memstore struct {
	grants       map[uuid.Array]grants.Grant
	grantSources map[string]struct{}
	lock         sync.Mutex
}

func NewMemstore() *Memstore {
	return &Memstore{
		grants:       map[uuid.Array]grants.Grant{},
		grantSources: map[string]struct{}{},
	}
}

func (m *Memstore) CreateGrant(ctx context.Context, g grants.Grant) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.grants[g.ID.Array()]; ok {
		return grants.ErrGrantAlreadyExists
	}

	if _, ok := m.grantSources[g.SourceType+"."+g.SourceID]; ok {
		return grants.ErrGrantSourceAlreadyUsed
	}

	m.grants[g.ID.Array()] = g
	m.grantSources[g.SourceType+"."+g.SourceID] = struct{}{}

	return nil
}

func (m *Memstore) ExchangeGrant(ctx context.Context, id uuid.UUID) (grants.Grant, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	grant, ok := m.grants[id.Array()]
	if !ok {
		return grants.Grant{}, grants.ErrGrantNotFound
	}

	if grant.Used {
		return grants.Grant{}, grants.ErrGrantAlreadyUsed
	}
	grant.Used = true
	m.grants[id.Array()] = grant

	return grant, nil
}
