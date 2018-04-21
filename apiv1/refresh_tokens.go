package apiv1

import (
	"context"
	"strings"

	"impractical.co/auth/grants"
	"impractical.co/auth/tokens"
)

func (a APIv1) issueTokens(ctx context.Context, grant grants.Grant) (Token, APIError) {
	// generate access first, so if there's a problem
	// the refresh token isn't just floating around, unused
	access, err := a.IssueAccessToken(ctx, grant)
	if err != nil {
		a.Log.WithError(err).Error("Error generating access token")
		return Token{}, serverError
	}

	refresh, err := a.IssueRefreshToken(ctx, grant)
	if err != nil {
		a.Log.WithError(err).Error("Error issuing refresh token")
		return Token{}, serverError
	}
	return Token{
		AccessToken:  access,
		TokenType:    "Bearer",
		ExpiresIn:    3600, // TODO(paddy): don't hardcode expiry
		RefreshToken: refresh,
		Scope:        strings.Join([]string(grant.Scopes), ","),
	}, APIError{}
}

type refreshTokenGranter struct {
	tokenVal string
	token    tokens.RefreshToken
	client   string
	deps     grants.Dependencies
}

func (r *refreshTokenGranter) Validate(ctx context.Context) APIError {
	token, err := r.deps.ValidateRefreshToken(ctx, r.tokenVal, r.client)
	if err != nil {
		if err == tokens.ErrInvalidToken {
			return invalidGrantError
		}
		r.deps.Log.WithError(err).Error("Error validating refresh token")
		return serverError
	}
	r.token = token
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
	return r.deps.UseRefreshToken(ctx, r.token.ID)
}

func (r *refreshTokenGranter) Redirects() bool {
	return false
}
