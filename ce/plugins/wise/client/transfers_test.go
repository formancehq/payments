package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Regression test: CreateTransfer must send Content-Type: application/json.
// Unlike CreateQuote/CreatePayout/CreateWebhook it did not, and nothing
// downstream (httpwrapper, transport) sets a default — Wise answered every
// transfer creation with 500 internal.server.error, so the connector's
// transfer initiation was broken against the real API.
func TestCreateTransferSetsContentType(t *testing.T) {
	t.Parallel()

	var gotContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": 2147689353,
			"targetAccount": 702499330,
			"quoteUuid": "482ef789-9f3d-41f0-9b7f-bc37c4d748be",
			"status": "incoming_payment_waiting",
			"created": "2026-07-02 12:38:02",
			"sourceCurrency": "EUR",
			"sourceValue": 0.05,
			"targetCurrency": "EUR",
			"targetValue": 0.05,
			"customerTransactionId": "ab0e0046-c368-41b6-a952-d3a126ac286d"
		}`))
	}))
	defer server.Close()

	c := newWithEndpoint("wise", "test-key", server.URL)

	transfer, err := c.CreateTransfer(context.Background(),
		Quote{ID: uuid.MustParse("482ef789-9f3d-41f0-9b7f-bc37c4d748be")}, 702499330, uuid.NewString())
	require.NoError(t, err)
	require.Equal(t, "application/json", gotContentType)
	require.Equal(t, uint64(2147689353), transfer.ID)
	require.False(t, transfer.CreatedAt.IsZero())
}
