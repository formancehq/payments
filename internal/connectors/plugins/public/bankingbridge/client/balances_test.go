package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/bankingbridge/client"
	"github.com/stretchr/testify/assert"
)

func TestGetAccountBalances(t *testing.T) {
	t.Parallel()
	expectedBalances := []client.Balance{
		{
			AccountReference: "acc1",
			AmountInMinors:   int64(1234),
			ReportedAt:       time.Now().Truncate(time.Millisecond).UTC(),
			ImportedAt:       time.Now().Truncate(time.Millisecond).UTC(),
			UpdatedAt:        time.Now().Truncate(time.Millisecond).UTC(),
		},
	}

	mockResponse := struct {
		Cursor struct {
			PageSize int64            `json:"pageSize"`
			Next     string           `json:"next"`
			Previous string           `json:"previous"`
			HasMore  bool             `json:"hasMore"`
			Data     []client.Balance `json:"data"`
		} `json:"cursor"`
	}{
		Cursor: struct {
			PageSize int64            `json:"pageSize"`
			Next     string           `json:"next"`
			Previous string           `json:"previous"`
			HasMore  bool             `json:"hasMore"`
			Data     []client.Balance `json:"data"`
		}{
			PageSize: 10,
			Next:     "next_cursor",
			HasMore:  true,
			Data:     expectedBalances,
		},
	}

	accessToken := "abcdefg"
	authServer := authServer(accessToken)
	defer authServer.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer "+accessToken, r.Header.Get("Authorization"))
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockResponse) //nolint: errcheck
	}))
	defer server.Close()

	c := client.New("bbs", "clientID", "clientSecret", authServer.URL, server.URL)
	balances, hasMore, nextCursor, err := c.GetAccountBalances(context.Background(), "", "", 10)
	assert.NoError(t, err)
	assert.Equal(t, expectedBalances, balances)
	assert.True(t, hasMore)
	assert.Equal(t, "next_cursor", nextCursor)
}
