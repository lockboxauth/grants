package googleid

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"github.com/ericchiang/oidc"
)

var (
	ErrInvalidAudience = errors.New("Invalid token audience.")
)

type Token struct {
	Iss   string `json:"iss"`
	Scope string `json:"scope,omitempty"`
	Aud   string `json:"aud"`
	Exp   int64  `json:"exp"`
	Iat   int64  `json:"iat"`
	Typ   string `json:"typ,omitempty"`

	Sub           string `json:"sub,omitempty"`
	Hd            string `json:"hd,omitempty"`
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	Name          string `json:"name,omitempty"`
	Picture       string `json:"picture,omitempty"`
	GivenName     string `json:"given_name,omitempty"`
	FamilyName    string `json:"family_name,omitempty"`
	Locale        string `json:"locale,omitempty"`

	source string
}

func Decode(payload string) (*Token, error) {
	s := strings.Split(payload, ".")
	if len(s) < 2 {
		return nil, errors.New("invalid token")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(s[1])
	if err != nil {
		return nil, err
	}
	t := &Token{}
	err = json.NewDecoder(bytes.NewBuffer(decoded)).Decode(t)
	if err != nil {
		return nil, err
	}
	t.source = payload
	return t, nil
}

func Verify(ctx context.Context, token string, clientIDs []string, verifier *oidc.IDTokenVerifier) error {
	tok, err := verifier.Verify(token)
	if err != nil {
		return err
	}
	for _, aud := range tok.Audience {
		for _, id := range clientIDs {
			if aud == id {
				return nil
			}
		}
	}
	return ErrInvalidAudience
}
