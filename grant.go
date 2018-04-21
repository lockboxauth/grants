package grants

//go:generate go-bindata -pkg migrations -o migrations/generated.go sql/

import (
	"context"
	"errors"
	"time"

	"impractical.co/auth/sessions"
	"impractical.co/auth/tokens"
	yall "yall.in"

	"github.com/hashicorp/go-uuid"
)

var (
	ErrGrantAlreadyUsed       = errors.New("grant already used, cannot be exchanged again")
	ErrGrantNotFound          = errors.New("grant not found")
	ErrGrantAlreadyExists     = errors.New("grant with that ID already exists")
	ErrGrantSourceAlreadyUsed = errors.New("grant source already used to generate a grant, cannot be used to create another grant")
)

type Grant struct {
	ID         string
	SourceType string
	SourceID   string
	CreatedAt  time.Time
	UsedAt     time.Time
	Scopes     []string
	ProfileID  string
	ClientID   string
	CreateIP   string
	UseIP      string
	Used       bool
}

type GrantUse struct {
	Grant string
	IP    string
	Time  time.Time
}

type Storer interface {
	CreateGrant(ctx context.Context, g Grant) error
	ExchangeGrant(ctx context.Context, g GrantUse) (Grant, error)
}

type Dependencies struct {
	Storer   Storer
	refresh  tokens.Dependencies
	sessions sessions.Dependencies
	Log      *yall.Logger
}

func FillGrantDefaults(grant Grant) (Grant, error) {
	res := grant
	if grant.ID == "" {
		id, err := uuid.GenerateUUID()
		if err != nil {
			return Grant{}, err
		}
		res.ID = id
	}
	if grant.CreatedAt.IsZero() {
		res.CreatedAt = time.Now()
	}
	return res, nil
}
