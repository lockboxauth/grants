package grants

//go:generate go-bindata -pkg migrations -o migrations/generated.go sql/

import (
	"context"
	"errors"
	"time"

	"github.com/apex/log"
	"github.com/pborman/uuid"
	"impractical.co/pqarrays"
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
	Scopes     pqarrays.StringArray
	ProfileID  string
	ClientID   string
	IP         string
	Used       bool
}

func (g Grant) GetSQLTableName() string {
	return "grants"
}

type Storer interface {
	CreateGrant(ctx context.Context, g Grant) error
	ExchangeGrant(ctx context.Context, id string) (Grant, error)
}

type Dependencies struct {
	Storer Storer
	Log    *log.Logger
}

func FillGrantDefaults(grant Grant) (Grant, error) {
	res := grant
	if grant.ID == "" {
		res.ID = uuid.NewRandom().String()
	}
	if grant.CreatedAt.IsZero() {
		res.CreatedAt = time.Now()
	}
	return res, nil
}
