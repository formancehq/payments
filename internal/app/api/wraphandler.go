package api

import (
	"net/http"

	"github.com/numary/go-libs/sharedauth"
)

func wrapHandler(useScopes bool, h http.Handler, scopes ...string) http.Handler {
	if !useScopes {
		return h
	}

	return sharedauth.NeedOneOfScopes(scopes...)(h)
}
