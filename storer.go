package grants

import (
	"context"
)

// Storer is the interface that Grants are persisted and used through.
type Storer interface {
	CreateGrant(ctx context.Context, g Grant) error
	ExchangeGrant(ctx context.Context, g GrantUse) (Grant, error)
	RevokeGrant(ctx context.Context, id string) (Grant, error)
	GetGrant(ctx context.Context, id string) (Grant, error)
	GetGrantBySource(ctx context.Context, sourceType, sourceID string) (Grant, error)
}
