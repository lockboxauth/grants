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
	Ancestors  []GrantAncestor `sql_column:"-"`
	CreatedAt  time.Time
	UsedAt     time.Time
	Scopes     pqarrays.StringArray
	AccountID  string
	ProfileID  string
	ClientID   string
	CreateIP   string
	UseIP      string
	Used       bool
	Revoked    bool
}

func (g Grant) AncestorIDs() []string {
	res := make([]string, 0, len(g.Ancestors))
	for _, anc := range g.Ancestors {
		res = append(res, anc.AncestorID)
	}
	return res
}

type GrantAncestor struct {
	GrantID    string
	AncestorID string
}

func (GrantAncestor) GetSQLTableName() string {
	return "grants_ancestors"
}

func ancestorsFromIDs(grantID string, ancestorIDs []string) []GrantAncestor {
	res := make([]GrantAncestor, 0, len(ancestorIDs))
	for _, anc := range ancestorIDs {
		res = append(res, GrantAncestor{
			GrantID:    grantID,
			AncestorID: anc,
		})
	}
	return res
}

// GetSQLTableName allows us to use Grant with
// pan.
func (Grant) GetSQLTableName() string {
	return "grants"
}

func fromPostgres(grant Grant) grants.Grant {
	return grants.Grant{
		ID:          grant.ID,
		SourceType:  grant.SourceType,
		SourceID:    grant.SourceID,
		AncestorIDs: grant.AncestorIDs(),
		CreatedAt:   grant.CreatedAt,
		UsedAt:      grant.UsedAt,
		Scopes:      []string(grant.Scopes),
		AccountID:   grant.AccountID,
		ProfileID:   grant.ProfileID,
		ClientID:    grant.ClientID,
		CreateIP:    grant.CreateIP,
		UseIP:       grant.UseIP,
		Used:        grant.Used,
		Revoked:     grant.Revoked,
	}
}

func toPostgres(grant grants.Grant) Grant {
	return Grant{
		ID:         grant.ID,
		SourceType: grant.SourceType,
		SourceID:   grant.SourceID,
		Ancestors:  ancestorsFromIDs(grant.ID, grant.AncestorIDs),
		CreatedAt:  grant.CreatedAt,
		UsedAt:     grant.UsedAt,
		Scopes:     pqarrays.StringArray(grant.Scopes),
		AccountID:  grant.AccountID,
		ProfileID:  grant.ProfileID,
		ClientID:   grant.ClientID,
		CreateIP:   grant.CreateIP,
		UseIP:      grant.UseIP,
		Used:       grant.Used,
		Revoked:    grant.Revoked,
	}
}
