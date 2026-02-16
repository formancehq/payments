package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/formancehq/payments/genericclient"
	"github.com/stretchr/testify/require"
)

func mockAPIClient(handler http.HandlerFunc) (*client, *httptest.Server) {
	server := httptest.NewServer(handler)

	configuration := genericclient.NewConfiguration()
	configuration.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	configuration.Servers[0].URL = server.URL

	return &client{apiClient: genericclient.NewAPIClient(configuration)}, server
}

// --- Payout tests ---

func TestCreatePayout_Success(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Format(time.RFC3339)
	expectedResp := PayoutResponse{
		Id:                   "payout_123",
		IdempotencyKey:       "ref_123",
		Amount:               "10000",
		Currency:             "USD/2",
		SourceAccountID:      "src_acc",
		DestinationAccountID: "dst_acc",
		Status:               "PENDING",
		CreatedAt:            now,
	}

	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/payouts", r.URL.Path)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer func() { _ = r.Body.Close() }()

		var req PayoutRequest
		require.NoError(t, json.Unmarshal(body, &req))
		require.Equal(t, "ref_123", req.IdempotencyKey)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(expectedResp)
	})
	defer server.Close()

	resp, err := c.CreatePayout(context.Background(), &PayoutRequest{
		IdempotencyKey:       "ref_123",
		Amount:               "10000",
		Currency:             "USD/2",
		SourceAccountID:      "src_acc",
		DestinationAccountID: "dst_acc",
	})
	require.NoError(t, err)
	require.Equal(t, expectedResp.Id, resp.Id)
	require.Equal(t, expectedResp.Status, resp.Status)
}

func TestCreatePayout_HTTPError(t *testing.T) {
	t.Parallel()

	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "bad request"}`))
	})
	defer server.Close()

	resp, err := c.CreatePayout(context.Background(), &PayoutRequest{IdempotencyKey: "ref_123"})
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "400")
}

func TestCreatePayout_InvalidJSON(t *testing.T) {
	t.Parallel()

	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid json`))
	})
	defer server.Close()

	resp, err := c.CreatePayout(context.Background(), &PayoutRequest{IdempotencyKey: "ref_123"})
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "unmarshal")
}

func TestCreatePayout_WithMetadata(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Format(time.RFC3339)
	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		defer func() { _ = r.Body.Close() }()

		var req PayoutRequest
		_ = json.Unmarshal(body, &req)
		require.Equal(t, "value1", req.Metadata["key1"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(PayoutResponse{
			Id: "payout_meta", CreatedAt: now, Status: "PENDING", Metadata: req.Metadata,
		})
	})
	defer server.Close()

	resp, err := c.CreatePayout(context.Background(), &PayoutRequest{
		IdempotencyKey: "ref_meta",
		Metadata:       map[string]string{"key1": "value1"},
	})
	require.NoError(t, err)
	require.Equal(t, "value1", resp.Metadata["key1"])
}

// --- Transfer tests ---

func TestCreateTransfer_Success(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Format(time.RFC3339)
	expectedResp := TransferResponse{
		Id:                   "transfer_123",
		IdempotencyKey:       "ref_456",
		Amount:               "50000",
		Currency:             "EUR/2",
		SourceAccountID:      "src_acc",
		DestinationAccountID: "dst_acc",
		Status:               "PENDING",
		CreatedAt:            now,
	}

	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/transfers", r.URL.Path)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer func() { _ = r.Body.Close() }()

		var req TransferRequest
		require.NoError(t, json.Unmarshal(body, &req))
		require.Equal(t, "ref_456", req.IdempotencyKey)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(expectedResp)
	})
	defer server.Close()

	desc := "Test transfer"
	resp, err := c.CreateTransfer(context.Background(), &TransferRequest{
		IdempotencyKey:       "ref_456",
		Amount:               "50000",
		Currency:             "EUR/2",
		SourceAccountID:      "src_acc",
		DestinationAccountID: "dst_acc",
		Description:          &desc,
	})
	require.NoError(t, err)
	require.Equal(t, expectedResp.Id, resp.Id)
	require.Equal(t, expectedResp.Status, resp.Status)
}

func TestCreateTransfer_HTTPError(t *testing.T) {
	t.Parallel()

	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "internal error"}`))
	})
	defer server.Close()

	resp, err := c.CreateTransfer(context.Background(), &TransferRequest{IdempotencyKey: "ref_456"})
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "500")
}

func TestCreateTransfer_InvalidJSON(t *testing.T) {
	t.Parallel()

	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	})
	defer server.Close()

	resp, err := c.CreateTransfer(context.Background(), &TransferRequest{IdempotencyKey: "ref_456"})
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "unmarshal")
}

func TestCreateTransfer_WithDescription(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Format(time.RFC3339)
	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		defer func() { _ = r.Body.Close() }()

		var req TransferRequest
		_ = json.Unmarshal(body, &req)
		require.NotNil(t, req.Description)
		require.Equal(t, "Payment for services", *req.Description)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(TransferResponse{
			Id: "transfer_desc", CreatedAt: now, Status: "PENDING", Description: req.Description,
		})
	})
	defer server.Close()

	desc := "Payment for services"
	resp, err := c.CreateTransfer(context.Background(), &TransferRequest{
		IdempotencyKey: "ref_desc",
		Description:    &desc,
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Description)
	require.Equal(t, desc, *resp.Description)
}

// --- Fetch tests (ListAccounts, GetBalances, ListBeneficiaries, ListTransactions) ---

func TestListAccounts_Success(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.URL.Path, "/accounts")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"id": "acc_1", "name": "Test Account", "currency": "USD", "createdAt": now.Format(time.RFC3339)},
		})
	})
	defer server.Close()

	accounts, err := c.ListAccounts(context.Background(), 1, 10, time.Time{})
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	require.Equal(t, "acc_1", accounts[0].Id)
}

func TestListAccounts_WithCreatedAtFrom(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		require.Contains(t, r.URL.RawQuery, "createdAtFrom")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
	})
	defer server.Close()

	accounts, err := c.ListAccounts(context.Background(), 1, 10, now.Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, accounts, 0)
}

func TestListAccounts_Error(t *testing.T) {
	t.Parallel()

	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "server error"}`))
	})
	defer server.Close()

	_, err := c.ListAccounts(context.Background(), 1, 10, time.Time{})
	require.Error(t, err)
}

func TestGetBalances_Success(t *testing.T) {
	t.Parallel()

	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.URL.Path, "/accounts/acc_123/balances")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"balances": []map[string]interface{}{{"currency": "USD", "amount": "1000"}},
		})
	})
	defer server.Close()

	balances, err := c.GetBalances(context.Background(), "acc_123")
	require.NoError(t, err)
	require.NotNil(t, balances)
}

func TestGetBalances_Error(t *testing.T) {
	t.Parallel()

	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "not found"}`))
	})
	defer server.Close()

	_, err := c.GetBalances(context.Background(), "acc_123")
	require.Error(t, err)
}

func TestListBeneficiaries_Success(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.URL.Path, "/beneficiaries")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"id": "ben_1", "name": "Test Beneficiary", "createdAt": now.Format(time.RFC3339)},
		})
	})
	defer server.Close()

	beneficiaries, err := c.ListBeneficiaries(context.Background(), 1, 10, time.Time{})
	require.NoError(t, err)
	require.Len(t, beneficiaries, 1)
	require.Equal(t, "ben_1", beneficiaries[0].Id)
}

func TestListBeneficiaries_Error(t *testing.T) {
	t.Parallel()

	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "server error"}`))
	})
	defer server.Close()

	_, err := c.ListBeneficiaries(context.Background(), 1, 10, time.Time{})
	require.Error(t, err)
}

func TestListTransactions_Success(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.URL.Path, "/transactions")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"id": "tx_1", "amount": "1000", "currency": "USD", "type": "PAYIN", "status": "SUCCEEDED",
				"createdAt": now.Format(time.RFC3339), "updatedAt": now.Format(time.RFC3339)},
		})
	})
	defer server.Close()

	transactions, err := c.ListTransactions(context.Background(), 1, 10, time.Time{})
	require.NoError(t, err)
	require.Len(t, transactions, 1)
	require.Equal(t, "tx_1", transactions[0].Id)
}

func TestListTransactions_Error(t *testing.T) {
	t.Parallel()

	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "server error"}`))
	})
	defer server.Close()

	_, err := c.ListTransactions(context.Background(), 1, 10, time.Time{})
	require.Error(t, err)
}

// --- Infrastructure tests ---

func TestNew_CreatesClient(t *testing.T) {
	t.Parallel()
	c := New("test-connector", "api-key-123", "https://api.example.com")
	require.NotNil(t, c)
}

func TestAPITransport_AddsAuthHeader(t *testing.T) {
	t.Parallel()

	var capturedHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	transport := &apiTransport{APIKey: "test-api-key", underlying: http.DefaultTransport}
	httpClient := &http.Client{Transport: transport}
	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	_, err := httpClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, "Bearer test-api-key", capturedHeader)
}

// --- Empty body / read error tests ---

func TestCreatePayout_ReadBodyError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	configuration := genericclient.NewConfiguration()
	configuration.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	configuration.Servers[0].URL = server.URL
	c := &client{apiClient: genericclient.NewAPIClient(configuration)}

	_, err := c.CreatePayout(context.Background(), &PayoutRequest{IdempotencyKey: "ref_123", Amount: "10000", Currency: "USD/2"})
	require.Error(t, err)
}

func TestCreateTransfer_ReadBodyError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	configuration := genericclient.NewConfiguration()
	configuration.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	configuration.Servers[0].URL = server.URL
	c := &client{apiClient: genericclient.NewAPIClient(configuration)}

	_, err := c.CreateTransfer(context.Background(), &TransferRequest{IdempotencyKey: "ref_123", Amount: "10000", Currency: "USD/2"})
	require.Error(t, err)
}

// --- Network error tests ---

type failingTransport struct{}

func (f *failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("network error: connection refused")
}

func TestCreatePayout_NetworkError(t *testing.T) {
	t.Parallel()

	configuration := genericclient.NewConfiguration()
	configuration.HTTPClient = &http.Client{Transport: &failingTransport{}}
	configuration.Servers[0].URL = "http://localhost:9999"
	c := &client{apiClient: genericclient.NewAPIClient(configuration)}

	resp, err := c.CreatePayout(context.Background(), &PayoutRequest{IdempotencyKey: "ref_123"})
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "failed to execute payout request")
}

func TestCreateTransfer_NetworkError(t *testing.T) {
	t.Parallel()

	configuration := genericclient.NewConfiguration()
	configuration.HTTPClient = &http.Client{Transport: &failingTransport{}}
	configuration.Servers[0].URL = "http://localhost:9999"
	c := &client{apiClient: genericclient.NewAPIClient(configuration)}

	resp, err := c.CreateTransfer(context.Background(), &TransferRequest{IdempotencyKey: "ref_123"})
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "failed to execute transfer request")
}

// --- Read response error tests ---

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("read error: broken pipe")
}

func (e *errorReader) Close() error { return nil }

type errorBodyTransport struct{ statusCode int }

func (e *errorBodyTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: e.statusCode, Body: &errorReader{}, Header: make(http.Header)}, nil
}

func TestCreatePayout_ReadResponseError(t *testing.T) {
	t.Parallel()

	configuration := genericclient.NewConfiguration()
	configuration.HTTPClient = &http.Client{Transport: &errorBodyTransport{statusCode: http.StatusOK}}
	configuration.Servers[0].URL = "http://localhost:9999"
	c := &client{apiClient: genericclient.NewAPIClient(configuration)}

	resp, err := c.CreatePayout(context.Background(), &PayoutRequest{IdempotencyKey: "ref_123"})
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "failed to read payout response")
}

func TestCreateTransfer_ReadResponseError(t *testing.T) {
	t.Parallel()

	configuration := genericclient.NewConfiguration()
	configuration.HTTPClient = &http.Client{Transport: &errorBodyTransport{statusCode: http.StatusOK}}
	configuration.Servers[0].URL = "http://localhost:9999"
	c := &client{apiClient: genericclient.NewAPIClient(configuration)}

	resp, err := c.CreateTransfer(context.Background(), &TransferRequest{IdempotencyKey: "ref_123"})
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "failed to read transfer response")
}
