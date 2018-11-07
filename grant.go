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
	// ErrGrantAlreadyUsed is returned when a grant is being used, but has already been used. This
	// usually indicates a replay attack.
	ErrGrantAlreadyUsed       = errors.New("grant already used, cannot be exchanged again")
	// ErrGrantNotFound is returned when a grant is being used or referenced, but does not exist in
	// the Storer being used. This usually indicates an invalid Grant is being presented.
	ErrGrantNotFound          = errors.New("grant not found")
	// ErrGrantAlreadyExists is returned when a grant is being stored in a Storer, but a grant with
	// the same ID alredy exists in that Storer. This usually indicates a programming error.
	ErrGrantAlreadyExists     = errors.New("grant with that ID already exists")
	// ErrGrantSourceAlreadyUsed is returned when a grant is being stored in a Storer, but the source
	// of the Grant has already been used in that Storer. This usually indicates a replay attack.
	ErrGrantSourceAlreadyUsed = errors.New("grant source already used to generate a grant, cannot be used to create another grant")
)

// Grant represents a user's authorization for the use of their account to some client.
type Grant struct {
	ID         string // a unique ID
	SourceType string // the type of the source used to identify the user
	SourceID   string // the ID of the source used to identify the user
	CreatedAt  time.Time // when the authorization was granted
	UsedAt     time.Time // when the authorization was exchanged for a session
	Scopes     []string // the scopes of access the user granted
	ProfileID  string // the unique ID representing the user account
	ClientID   string // the client access was granted to
	CreateIP   string // the IP the user granted access from
	UseIP      string // the IP the access was exchanged for a session from
	Used       bool // whether the access has been exchanged for a session or not
}

// GrantUse represents the exchange of a Grant for a session.
type GrantUse struct {
	Grant string // the ID of the grant that was exchanged
	IP    string // the IP address the exchange was initiated from
	Time  time.Time // the time the exchange happened
}

// Storer is the interface that Grants are persisted and used through.
type Storer interface {
	CreateGrant(ctx context.Context, g Grant) error
	ExchangeGrant(ctx context.Context, g GrantUse) (Grant, error)
}

// Dependencies bundles together the information needed to run the service.
type Dependencies struct {
	Storer   Storer // the Storer to store Grants in
	refresh  tokens.Dependencies // the service to create refresh tokens with
	sessions sessions.Dependencies // the service to create sessions with
	Log      *yall.Logger // the logger to use
}

// FillGrantDefaults sets any unset fields of Grant that have a default value.
// Fields that are set and fields that have no default value are not modified.
// The original Grant is not modified; a shallow copy is made and modified, then
// returned.
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
