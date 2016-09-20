package apiv1

import (
	"net/http"

	"darlinggo.co/trout"
)

func (a APIv1) Server(prefix string) http.Handler {
	var router trout.Router
	router.SetPrefix(prefix)

	router.Endpoint("/token").Methods("POST").Handler(http.HandlerFunc(a.handleAccessTokenRequest))

	return router
}
