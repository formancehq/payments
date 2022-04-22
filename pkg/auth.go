package payment

import (
	"errors"
	"github.com/numary/go-libs/sharedauth/sharedauthjwt"
	"net/http"
	"strings"
)

type Method interface {
	IsMatching(r *http.Request) bool
	Check(r *http.Request) error
}

type httpBasicMethod struct {
	credentials map[string]string
}

func (h httpBasicMethod) IsMatching(r *http.Request) bool {
	return strings.HasPrefix(
		strings.ToLower(r.Header.Get("Authorization")),
		"basic",
	)
}

func (h httpBasicMethod) Check(r *http.Request) error {
	username, password, ok := r.BasicAuth()
	if !ok {
		return errors.New("malformed basic")
	}
	if username == "" {
		return errors.New("malformed basic")
	}
	if h.credentials[username] != password {
		return errors.New("invalid credentials")
	}
	return nil
}

func NewHTTPBasicMethod(credentials map[string]string) *httpBasicMethod {
	return &httpBasicMethod{
		credentials: credentials,
	}
}

var _ Method = &httpBasicMethod{}

type httpBearerMethod struct {
	authUrl string
	client  *http.Client
}

func (h httpBearerMethod) IsMatching(r *http.Request) bool {
	return strings.HasPrefix(
		strings.ToLower(r.Header.Get("Authorization")),
		"bearer",
	)
}

func (h httpBearerMethod) Check(r *http.Request) error {
	err := sharedauthjwt.CheckTokenWithAuth(h.client, h.authUrl, r)
	if err != nil {
		return err
	}
	return nil
}

var _ Method = &httpBearerMethod{}

func NewHttpBearerMethod(client *http.Client, authUrl string) *httpBearerMethod {
	return &httpBearerMethod{
		authUrl: authUrl,
		client:  client,
	}
}

func AuthMiddleware(methods ...Method) func(handler http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ok := false
			for _, m := range methods {
				if m.IsMatching(r) {
					err := m.Check(r)
					if err != nil {
						w.WriteHeader(http.StatusForbidden)
						return
					}
					ok = true
					break
				}
			}
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			h.ServeHTTP(w, r)
		})
	}
}
