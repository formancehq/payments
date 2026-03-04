package client_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

func authServer(accessToken string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		var res = struct {
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
		}{
			AccessToken: accessToken,
			TokenType:   "Bearer",
		}
		json.NewEncoder(w).Encode(&res) //nolint: errcheck
	}))
}
