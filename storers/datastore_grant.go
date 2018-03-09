package storers

import (
	"time"

	"impractical.co/auth/grants"
)

type datastoreGrant struct {
	ID         string
	SourceType string
	SourceID   string
	CreatedAt  time.Time
	Scopes     []string
	ProfileID  string
	ClientID   string
	IP         string
	Used       bool
}

func fromDatastore(g datastoreGrant) grants.Grant {
	return grants.Grant(g)
}

func toDatastore(g grants.Grant) datastoreGrant {
	return datastoreGrant(g)
}

type datastoreSourceRecord struct {
	ID         string
	SourceType string
	SourceID   string
	CreatedAt  time.Time
}
