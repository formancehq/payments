package payment

import (
	"context"
	"github.com/gorilla/mux"
	"net/http"
)

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

func Recovery(reporter func(ctx context.Context, e interface{})) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if e := recover(); e != nil {
					w.WriteHeader(http.StatusInternalServerError)
					reporter(r.Context(), e)
				}
			}()
			h.ServeHTTP(w, r)
		})
	}
}
