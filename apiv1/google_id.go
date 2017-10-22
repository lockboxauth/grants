package apiv1

import (
	"context"
	"strconv"
	"strings"

	"github.com/apex/log"
	"github.com/coreos/go-oidc"

	"impractical.co/auth/accounts"
	"impractical.co/auth/grants"
	"impractical.co/googleid"
)

type googleIDGranter struct {
	tokenVal     string                // the token
	client       string                // the client that created the token
	gClients     []string              // the Google clients that the token must be for
	oidcVerifier *oidc.IDTokenVerifier // the verifier that we can use to verify tokens
	accounts     accounts.Storer       // the Storer that grants access to accounts data

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
	account, err := g.accounts.Get(ctx, strings.ToLower(token.Email))
	if err != nil {
		g.log.WithError(err).WithField("email", token.Email).Error("Error retriving account")
		return serverError
	}
	if account.ProfileID == "" {
		g.log.WithError(err).WithField("email", token.Email).Debug("Empty ProfileID")
		return invalidGrantError
	}
	g.userID = account.ProfileID
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
