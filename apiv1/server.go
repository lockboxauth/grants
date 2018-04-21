package apiv1

import (
	"net/http"

	"darlinggo.co/trout"
	yall "yall.in"
)

func (a APIv1) contextLogger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := a.Log.WithRequest(r).WithField("endpoint", r.Header.Get("Trout-Pattern"))
		r = r.WithContext(yall.InContext(r.Context(), log))
		log.Debug("serving request")
		h.ServeHTTP(w, r)
	})
}

func (a APIv1) Server(prefix string) http.Handler {
	var router trout.Router
	router.SetPrefix(prefix)

	router.Endpoint("/token").Methods("POST").Handler(a.contextLogger(http.HandlerFunc(a.handleAccessTokenRequest)))

	return router
}
