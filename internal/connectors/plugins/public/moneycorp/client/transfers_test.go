package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/stretchr/testify/require"
)

func TestInitiateTransfer_404DoesNotPanic(t *testing.T) {
	t.Parallel()

	// Moneycorp answers 404 when the source account is unknown. The error
	// checker maps it to a nil error, so the body is empty and Transfer is
	// nil. The client must surface a non-retryable error, not panic.
	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors":[{"detail":"account not found"}]}`))
	})
	defer server.Close()

	resp, err := c.InitiateTransfer(context.Background(), &TransferRequest{SourceAccountID: "123"})
	require.Error(t, err)
	require.Nil(t, resp)
	require.ErrorIs(t, err, models.ErrInvalidRequest)
}

func TestInitiateTransfer_EmptyBody(t *testing.T) {
	t.Parallel()

	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	defer server.Close()

	resp, err := c.InitiateTransfer(context.Background(), &TransferRequest{SourceAccountID: "123"})
	require.Error(t, err)
	require.Nil(t, resp)
	require.ErrorIs(t, err, models.ErrInvalidRequest)
	require.Contains(t, err.Error(), "empty response")
}

func TestInitiateTransfer_Success(t *testing.T) {
	t.Parallel()

	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(transferResponse{Transfer: &TransferResponse{ID: "tr_1"}})
	})
	defer server.Close()

	resp, err := c.InitiateTransfer(context.Background(), &TransferRequest{SourceAccountID: "123"})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, "tr_1", resp.ID)
}

func TestGetTransfer_404DoesNotPanic(t *testing.T) {
	t.Parallel()

	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{}`))
	})
	defer server.Close()

	resp, err := c.GetTransfer(context.Background(), "123", "tr_1")
	require.Error(t, err)
	require.Nil(t, resp)
	require.ErrorIs(t, err, models.ErrInvalidRequest)
}
