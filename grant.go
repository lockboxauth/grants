package grants

//go:generate go-bindata -pkg $GOPACKAGE -o migrations.go sql/

import (
	"errors"
	"time"

	"code.impractical.co/pqarrays"
	"github.com/pborman/uuid"
	"golang.org/x/net/context"
)

var (
	ErrGrantAlreadyUsed       = errors.New("grant already used, cannot be exchanged again")
	ErrGrantNotFound          = errors.New("grant not found")
	ErrGrantAlreadyExists     = errors.New("grant with that ID already exists")
	ErrGrantSourceAlreadyUsed = errors.New("grant source already used to generate a grant, cannot be used to create another grant")
)

type Grant struct {
	ID         uuid.UUID
	SourceType string
	SourceID   string
	CreatedAt  time.Time
	Scopes     pqarrays.StringArray
	ProfileID  string
	ClientID   string
	IP         string
	Used       bool
}

type Storer interface {
	CreateGrant(ctx context.Context, g Grant) error
	ExchangeGrant(ctx context.Context, id uuid.UUID) (Grant, error)
}

func FillGrantDefaults(grant Grant) (Grant, error) {
	res := grant
	if grant.ID == nil {
		res.ID = uuid.NewRandom()
	}
	if grant.CreatedAt.IsZero() {
		res.CreatedAt = time.Now()
	}
	return res, nil
}
