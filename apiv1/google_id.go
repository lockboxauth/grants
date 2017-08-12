package apiv1

import (
	"context"
	"strconv"

	"github.com/apex/log"
	"github.com/ericchiang/oidc"

	"code.impractical.co/googleid"
	"code.impractical.co/grants"
)

type googleIDGranter struct {
	tokenVal     string                // the token
	client       string                // the client that created the token
	gClients     []string              // the clients that the token must be for
	oidcVerifier *oidc.IDTokenVerifier // the verifier that we can use to verify tokens

	// set by Validate and here so Grant can use them
	userID string
	token  *googleid.Token

	log *log.Logger
}

func (g *googleIDGranter) Validate(ctx context.Context) APIError {
	token, err := googleid.Decode(g.tokenVal)
	if err != nil {
		g.log.WithError(err).Debug("Error decoding ID token")
		return invalidGrantError
	}
	err = googleid.Verify(ctx, g.tokenVal, g.gClients, g.oidcVerifier)
	if err != nil {
		g.log.WithError(err).Debug("Error verifying ID token")
		return invalidGrantError
	}
	g.token = token
	// TODO(paddy): retrieve account based on google ID's email
	// TODO(paddy): set g.userID to the profile ID associated with that account
	return APIError{}
}

func (g *googleIDGranter) Grant(ctx context.Context, scopes []string) grants.Grant {
	return grants.Grant{
		SourceType: "google_id",
		SourceID:   g.token.Iss + ":" + g.token.Sub + ":" + strconv.FormatInt(g.token.Iat, 10),
		ProfileID:  g.userID,
		ClientID:   g.client,
		Scopes:     scopes,
	}
}

func (g *googleIDGranter) Granted(ctx context.Context) error {
	return nil
}

func (g *googleIDGranter) Redirects() bool {
	return false
}
