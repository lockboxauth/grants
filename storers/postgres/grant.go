package postgres

import (
	"time"

	"impractical.co/pqarrays"

	"lockbox.dev/grants"
)

// Grant is a representation of a Grant
// suitable for storage in our Storer.
type Grant struct {
	ID         string
	SourceType string
	SourceID   string
	CreatedAt  time.Time
	UsedAt     time.Time
	Scopes     pqarrays.StringArray
	AccountID  string
	ProfileID  string
	ClientID   string
	CreateIP   string
	UseIP      string
	Used       bool
}

// GetSQLTableName allows us to use Grant with
// pan.
func (Grant) GetSQLTableName() string {
	return "grants"
}

func fromPostgres(grant Grant) grants.Grant {
	return grants.Grant{
		ID:         grant.ID,
		SourceType: grant.SourceType,
		SourceID:   grant.SourceID,
		CreatedAt:  grant.CreatedAt,
		UsedAt:     grant.UsedAt,
		Scopes:     []string(grant.Scopes),
		AccountID:  grant.AccountID,
		ProfileID:  grant.ProfileID,
		ClientID:   grant.ClientID,
		CreateIP:   grant.CreateIP,
		UseIP:      grant.UseIP,
		Used:       grant.Used,
	}
}

func toPostgres(grant grants.Grant) Grant {
	return Grant{
		ID:         grant.ID,
		SourceType: grant.SourceType,
		SourceID:   grant.SourceID,
		CreatedAt:  grant.CreatedAt,
		UsedAt:     grant.UsedAt,
		Scopes:     pqarrays.StringArray(grant.Scopes),
		AccountID:  grant.AccountID,
		ProfileID:  grant.ProfileID,
		ClientID:   grant.ClientID,
		CreateIP:   grant.CreateIP,
		UseIP:      grant.UseIP,
		Used:       grant.Used,
	}
}
