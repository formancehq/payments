package bridge

import (
	"github.com/numary/go-libs/sharedauth"
	"net/http"
)

func WrapHandler(useScopes bool, h http.Handler, scopes ...string) http.Handler {
	if !useScopes {
		return h
	}
	return sharedauth.NeedOneOfScopes(scopes...)(h)
}
