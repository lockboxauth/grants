package postgres

import (
	"time"

	"impractical.co/auth/grants"
	"impractical.co/pqarrays"
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
	ProfileID  string
	ClientID   string
	CreateIP   string
	UseIP      string
	Used       bool
}

// GetSQLTableName allows us to use Grant with
// pan.
func (g Grant) GetSQLTableName() string {
	return "grants"
}

func fromPostgres(g Grant) grants.Grant {
	return grants.Grant{
		ID:         g.ID,
		SourceType: g.SourceType,
		SourceID:   g.SourceID,
		CreatedAt:  g.CreatedAt,
		UsedAt:     g.UsedAt,
		Scopes:     []string(g.Scopes),
		ProfileID:  g.ProfileID,
		ClientID:   g.ClientID,
		CreateIP:   g.CreateIP,
		UseIP:      g.UseIP,
		Used:       g.Used,
	}
}

func toPostgres(g grants.Grant) Grant {
	return Grant{
		ID:         g.ID,
		SourceType: g.SourceType,
		SourceID:   g.SourceID,
		CreatedAt:  g.CreatedAt,
		UsedAt:     g.UsedAt,
		Scopes:     pqarrays.StringArray(g.Scopes),
		ProfileID:  g.ProfileID,
		ClientID:   g.ClientID,
		CreateIP:   g.CreateIP,
		UseIP:      g.UseIP,
		Used:       g.Used,
	}
}
