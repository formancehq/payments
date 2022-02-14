package payment

import (
	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"github.com/numary/go-libs-cloud/pkg/auth"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net/http"
	"runtime/debug"
	"strings"
)

func CheckOrganizationAccess(req *http.Request, name string) error {
	jwtString := req.Header.Get("Authorization")
	if !strings.HasPrefix(strings.ToLower(jwtString), "bearer ") {
		return auth.ErrAuthorizationHeaderNotFound
	}
	tokenString := jwtString[len("bearer "):]

	payload, _, err := new(jwt.Parser).ParseUnverified(tokenString, &auth.ClaimStruct{})
	if err != nil {
		return errors.Wrap(err, "parsing jwt token")
	}
	for _, s := range payload.Claims.(*auth.ClaimStruct).Organizations {
		if s.Name == name {
			return nil
		}
	}
	return auth.ErrAccessDenied
}

func CheckOrganizationAccessMiddleware() func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := CheckOrganizationAccess(r, mux.Vars(r)["organizationId"])
			switch err {
			case auth.ErrAccessDenied:
				w.WriteHeader(http.StatusForbidden)
				return
			case nil:
			default:
				w.WriteHeader(http.StatusUnauthorized)
			}
			h.ServeHTTP(w, r)
		})
	}
}

func ConfigureAuthMiddleware(m *mux.Router, middlewares ...mux.MiddlewareFunc) *mux.Router {
	err := m.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		path, err := route.GetPathTemplate()
		if err != nil {
			return mux.SkipRouter
		}
		if path == "/organizations/{organizationId}" {
			router.Use(middlewares...)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return m
}

func Recovery(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				logrus.Errorln(e)
				debug.PrintStack()
			}
		}()
		h.ServeHTTP(w, r)
	})
}
