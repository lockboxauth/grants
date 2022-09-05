package grants

import (
	"errors"
	"time"

	uuid "github.com/hashicorp/go-uuid"
)

var (
	// ErrGrantAlreadyUsed is returned when a grant is being used, but has already been used. This
	// usually indicates a replay attack.
	ErrGrantAlreadyUsed = errors.New("grant already used, cannot be exchanged again")
	// ErrGrantRevoked is returned when a grant is being used, but has been
	// revoked. This usually indicates a leaked credential being exploited.
	ErrGrantRevoked = errors.New("grant was revoked, cannot be exchanged")
	// ErrGrantNotFound is returned when a grant is being used or referenced, but does not exist in
	// the Storer being used. This usually indicates an invalid Grant is being presented.
	ErrGrantNotFound = errors.New("grant not found")
	// ErrGrantAlreadyExists is returned when a grant is being stored in a Storer, but a grant with
	// the same ID alredy exists in that Storer. This usually indicates a programming error.
	ErrGrantAlreadyExists = errors.New("grant with that ID already exists")
	// ErrGrantSourceAlreadyUsed is returned when a grant is being stored in a Storer, but the source
	// of the Grant has already been used in that Storer. This usually indicates a replay attack.
	ErrGrantSourceAlreadyUsed = errors.New("grant source already used to generate a grant, cannot be used to create another grant")
)

// Grant represents a user's authorization for the use of their account to some client.
type Grant struct {
	ID          string    // a unique ID
	SourceType  string    // the type of the source used to identify the user
	SourceID    string    // the ID of the source used to identify the user; should be unique across grants
	AncestorIDs []string  // the IDs of any Grants that led to the creation of this grant, e.g. through refresh
	CreatedAt   time.Time // when the authorization was granted
	UsedAt      time.Time // when the authorization was exchanged for a session
	Scopes      []string  // the scopes of access the user granted
	AccountID   string    // the ID of the account that was used to grant access
	ProfileID   string    // the unique ID representing the user
	ClientID    string    // the client access was granted to
	CreateIP    string    // the IP the user granted access from
	UseIP       string    // the IP the access was exchanged for a session from
	Used        bool      // whether the access has been exchanged for a session or not
	Revoked     bool      // whether the grant has been manually revoked or not
}

// GrantUse represents the exchange of a Grant for a session.
type GrantUse struct {
	Grant string    // the ID of the grant that was exchanged
	IP    string    // the IP address the exchange was initiated from
	Time  time.Time // the time the exchange happened
}

// Dependencies bundles together the information needed to run the service.
type Dependencies struct {
	Storer Storer // the Storer to store Grants in
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
