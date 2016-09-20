package apiv1

import (
	"context"
	"log"
	"time"

	"code.impractical.co/grants"
	refresh "code.impractical.co/tokens/client"
)

func (a APIv1) issueTokens(ctx context.Context, grant grants.Grant) (Token, APIError) {
	// TODO(paddy): exchange grant for refresh token and access token
	return Token{}, APIError{}
}

func (a APIv1) issueRefresh(ctx context.Context, grant grants.Grant) (string, error) {
	t := refresh.Token{
		CreatedAt:   time.Now(),
		CreatedFrom: grant.ID.String(),
		Scopes:      grant.Scopes,
		ProfileID:   grant.ProfileID,
		ClientID:    grant.ClientID,
	}
	t, errs := a.Tokens.Insert(ctx, t)
	if len(errs) > 0 {
		return "", errs[0] // TODO(paddy): stop dropping errors
	}
	return refresh.Build(t), nil
}

type refreshTokenGranter struct {
	tokenVal string
	token    refresh.Token
	client   string
	manager  refresh.Manager
	log      *log.Logger
}

func (r *refreshTokenGranter) Validate(ctx context.Context) APIError {
	tokenID, _, err := refresh.Break(r.tokenVal)
	if err != nil {
		return invalidGrantError
	}
	errs := r.manager.Validate(ctx, r.tokenVal)
	for _, err := range errs {
		if err == refresh.ErrInvalidTokenString {
			return invalidGrantError
		}
		r.log.Printf("Error validating refresh token: %+v\n", err)
	}
	if len(errs) > 0 {
		return serverError
	}
	r.token, errs = r.manager.Get(ctx, tokenID)
	if len(errs) > 0 {
		r.log.Printf("Error retrieving refresh token: %+v\n", errs)
		return serverError
	}
	if r.token.ClientID != r.client {
		return invalidGrantError
	}
	return APIError{}
}

func (r *refreshTokenGranter) Grant(ctx context.Context, scopes []string) grants.Grant {
	return grants.Grant{
		SourceType: "refresh_token",
		SourceID:   r.token.ID,
		Scopes:     r.token.Scopes,
		ProfileID:  r.token.ProfileID,
		ClientID:   r.token.ClientID,
	}
}

func (r *refreshTokenGranter) Granted(ctx context.Context) error {
	errs := r.manager.Use(ctx, r.token.ID)
	if len(errs) > 0 {
		return errs[0] // TODO(paddy): stop dropping errors
	}
	return nil
}

func (r *refreshTokenGranter) Redirects() bool {
	return false
}
