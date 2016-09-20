package apiv1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"code.impractical.co/grants"
	refresh "code.impractical.co/tokens/client"
)

var (
	serverError       = APIError{Error: "server_error", Code: http.StatusInternalServerError}
	invalidGrantError = APIError{Error: "invalid_grant", Code: http.StatusBadRequest}
)

type APIv1 struct {
	grants.Dependencies
	Tokens refresh.Manager
}

type APIError struct {
	Error string `json:"error"`
	Code  int    `json:"-"`
}

func (a APIError) IsZero() bool {
	return a.Error == ""
}

type granter interface {
	Validate(ctx context.Context) APIError
	Grant(ctx context.Context, scopes []string) grants.Grant
	Granted(ctx context.Context) error
	Redirects() bool
}

func (a APIv1) returnError(redirect bool, w http.ResponseWriter, r *http.Request, apiErr APIError) {
	if redirect {
		// TODO(paddy): actually redirect the user
		return
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(apiErr.Code)
	enc := json.NewEncoder(w)
	err := enc.Encode(apiErr)
	if err != nil {
		a.Log.Printf("Error writing response: %+v\n", err)
	}
}

func returnToken(redirect bool, w http.ResponseWriter, r *http.Request, token Token) {
	if redirect {
		// TODO(paddy): actually redirect the user
		return
	}
}

func getClientCredentials(r *http.Request) (id, secret, redirect string) {
	id = r.URL.Query().Get("client_id")
	redirect = r.URL.Query().Get("redirect_uri")
	if id != "" {
		return id, secret, redirect
	}
	redirect = ""
	var ok bool
	id, secret, ok = r.BasicAuth()
	if ok {
		return id, secret, redirect
	}
	id = r.PostFormValue("client_id")
	secret = r.PostFormValue("client_secret")
	return id, secret, redirect
}

func (a APIv1) validateClientCredentials(ctx context.Context, clientID, clientSecret, redirectURI string) APIError {
	if clientID == "" {
		// error
	}
	if clientSecret != "" && redirectURI != "" {
		// error
	}
	// TODO(paddy): retrieve client, and validate the credentials
	return APIError{}
}

func (a APIv1) checkScopes(ctx context.Context, clientID string, scopes []string) ([]string, APIError) {
	var results []string
	// TODO(paddy): if scopes is empty, populate it with a default set
	// TODO(paddy): if scopes contains any scopes the client can't use, remove them
	return results, APIError{}
}

func (a APIv1) createGrant(ctx context.Context, grant grants.Grant) APIError {
	grant, err := grants.FillGrantDefaults(grant)
	if err != nil {
		a.Log.Printf("Error filling grant defaults: %+v\n", err)
		return serverError
	}
	err = a.Storer.CreateGrant(ctx, grant)
	if err != nil {
		a.Log.Printf("Error creating grant: %+v\n", err)
		return serverError
	}
	return APIError{}
}

func (a APIv1) getGranter(values url.Values, clientID string) granter {
	switch values.Get("grant_type") {
	case "refresh_token":
		return &refreshTokenGranter{
			tokenVal: values.Get("refresh_token"),
			client:   clientID,
			manager:  a.Tokens,
			log:      a.Log,
		}
		/*case "password":
		return credentialsGranter{
			username: values.Get("username"),
			password: values.Get("password"),
			client:   clientID,
		}*/
	}
	var g granter
	return g
}

func (a APIv1) handleAccessTokenRequest(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		a.Log.Printf("Error parsing form: %+v\n", err)
		a.returnError(false, w, r, serverError)
		return
	}
	clientID, clientSecret, redirectURI := getClientCredentials(r)
	// wait until we know whether or not to redirect to handle client error
	clientErr := a.validateClientCredentials(r.Context(), clientID, clientSecret, redirectURI)
	g := a.getGranter(r.PostForm, clientID)
	if g == nil {
		a.returnError(false, w, r, APIError{Error: "unsupported_grant_type", Code: http.StatusBadRequest})
		return
	}
	if !clientErr.IsZero() {
		a.returnError(g.Redirects(), w, r, clientErr)
		return
	}
	apiErr := g.Validate(r.Context())
	if !apiErr.IsZero() {
		a.returnError(g.Redirects(), w, r, apiErr)
		return
	}
	scopes := strings.Split(r.FormValue("scope"), " ")
	grant := g.Grant(r.Context(), scopes)
	grant.Scopes, apiErr = a.checkScopes(r.Context(), grant.ClientID, grant.Scopes)
	if !apiErr.IsZero() {
		a.returnError(g.Redirects(), w, r, apiErr)
		return
	}
	grant.IP = getIP(r)
	apiErr = a.createGrant(r.Context(), grant)
	if !apiErr.IsZero() {
		a.returnError(g.Redirects(), w, r, apiErr)
		return
	}
	token, apiErr := a.issueTokens(r.Context(), grant)
	if !apiErr.IsZero() {
		a.returnError(g.Redirects(), w, r, apiErr)
		return
	}
	err = g.Granted(r.Context())
	if err != nil {
		a.Log.Printf("Error calling Granted for %+v: %+v\n", g, err)
	}
	returnToken(g.Redirects(), w, r, token)
}
