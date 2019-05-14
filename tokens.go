package grants

import (
	"context"
	"time"

	uuid "github.com/hashicorp/go-uuid"
	"impractical.co/auth/sessions"
	"impractical.co/auth/tokens"
	yall "yall.in"
)

// TODO: move this to the oauth2 package, it has no business being in this package

// IssueRefreshToken creates a Refresh Token and stores it in the service indicated by
// `refresh` on `d`. It fills the token with the appropriate values from `grant`, sets
// any unset defaults, and stores the token before returning it.
func (d Dependencies) IssueRefreshToken(ctx context.Context, grant Grant) (string, error) {
	t, err := tokens.FillTokenDefaults(tokens.RefreshToken{
		CreatedFrom: grant.ID,
		Scopes:      grant.Scopes,
		ProfileID:   grant.ProfileID,
		ClientID:    grant.ClientID,
	})
	if err != nil {
		return "", err
	}
	token, err := d.Refresh.CreateJWT(ctx, t)
	if err != nil {
		return "", err
	}
	err = d.Refresh.Storer.CreateToken(ctx, t)
	if err != nil {
		return "", err
	}
	return token, nil
}

// ValidateRefreshToken verifies that a refresh token is valid and for the specified
// client, returning the struct representation of valid tokens.
func (d Dependencies) ValidateRefreshToken(ctx context.Context, token, client string) (tokens.RefreshToken, error) {
	tok, err := d.Refresh.Validate(ctx, token)
	if err != nil {
		return tokens.RefreshToken{}, err
	}
	if tok.ClientID != client {
		yall.FromContext(ctx).WithField("client_id", client).WithField("desired_id", tok.ClientID).Debug("Client tried to use other client's refresh token.")
		return tokens.RefreshToken{}, tokens.ErrInvalidToken
	}
	return tok, nil
}

// UseRefreshToken marks a refresh token as used, making it so the token cannot be
// reused.
func (d Dependencies) UseRefreshToken(ctx context.Context, tokenID string) error {
	err := d.Refresh.Storer.UseToken(ctx, tokenID)
	if err != nil && err != tokens.ErrTokenUsed {
		yall.FromContext(ctx).WithField("token", tokenID).WithError(err).Error("Error using token.")
		return err
	}
	if err == tokens.ErrTokenUsed {
		return err
	}
	return nil
}

// IssueAccessToken creates a new access token from a Grant, filling in the values
// appropriately.
func (d Dependencies) IssueAccessToken(ctx context.Context, grant Grant) (string, error) {
	id, err := uuid.GenerateUUID()
	if err != nil {
		return "", err
	}
	return d.Sessions.CreateJWT(ctx, sessions.AccessToken{
		ID:          id,
		CreatedFrom: grant.ID,
		Scopes:      grant.Scopes,
		ProfileID:   grant.ProfileID,
		ClientID:    grant.ClientID,
		CreatedAt:   time.Now(),
	})
}
