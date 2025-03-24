package testserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
)

// StackServer handles requests from the SDK and forwards them to our test server
func StackServer(destinationUrl string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/payments")
		redirectUrl, err := url.JoinPath(destinationUrl, path)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf(`{"error":"%s"}`, err)))
			return
		}
		http.Redirect(w, r, redirectUrl, http.StatusPermanentRedirect)
	}))
}
