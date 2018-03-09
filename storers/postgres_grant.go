package storers

import (
	"time"

	"impractical.co/auth/grants"
	"impractical.co/pqarrays"
)

type postgresGrant struct {
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

func (p postgresGrant) GetSQLTableName() string {
	return "grants"
}

func fromPostgres(g postgresGrant) grants.Grant {
	return grants.Grant{
		ID:         g.ID,
		SourceType: g.SourceType,
		SourceID:   g.SourceID,
		CreatedAt:  g.CreatedAt,
		Scopes:     []string(g.Scopes),
		ProfileID:  g.ProfileID,
		ClientID:   g.ClientID,
		IP:         g.IP,
		Used:       g.Used,
	}
}

func toPostgres(g grants.Grant) postgresGrant {
	return postgresGrant{
		ID:         g.ID,
		SourceType: g.SourceType,
		SourceID:   g.SourceID,
		CreatedAt:  g.CreatedAt,
		Scopes:     pqarrays.StringArray(g.Scopes),
		ProfileID:  g.ProfileID,
		ClientID:   g.ClientID,
		IP:         g.IP,
		Used:       g.Used,
	}
}
