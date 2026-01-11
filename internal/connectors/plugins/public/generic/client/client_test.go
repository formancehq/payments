package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/formancehq/payments/genericclient"
	"github.com/stretchr/testify/require"
)

// mockAPIClient creates a test client with a mock HTTP server
func mockAPIClient(handler http.HandlerFunc) (*client, *httptest.Server) {
	server := httptest.NewServer(handler)

	configuration := genericclient.NewConfiguration()
	configuration.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	configuration.Servers[0].URL = server.URL

	return &client{apiClient: genericclient.NewAPIClient(configuration)}, server
}

// TestCreatePayout tests the CreatePayout client method
func TestCreatePayout_Success(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Format(time.RFC3339)
	expectedResp := PayoutResponse{
		Id:                   "payout_123",
		IdempotencyKey:       "ref_123",
		Amount:               "100.00",
		Currency:             "USD",
		SourceAccountId:      "src_acc",
		DestinationAccountId: "dst_acc",
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
		err = json.Unmarshal(body, &req)
		require.NoError(t, err)
		require.Equal(t, "ref_123", req.IdempotencyKey)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(expectedResp)
	})
	defer server.Close()

	req := &PayoutRequest{
		IdempotencyKey:       "ref_123",
		Amount:               "100.00",
		Currency:             "USD",
		SourceAccountId:      "src_acc",
		DestinationAccountId: "dst_acc",
	}

	resp, err := c.CreatePayout(context.Background(), req)
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

	req := &PayoutRequest{IdempotencyKey: "ref_123"}
	resp, err := c.CreatePayout(context.Background(), req)
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

	req := &PayoutRequest{IdempotencyKey: "ref_123"}
	resp, err := c.CreatePayout(context.Background(), req)
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "unmarshal")
}

// TestGetPayoutStatus tests the GetPayoutStatus client method
func TestGetPayoutStatus_Success(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Format(time.RFC3339)
	expectedResp := PayoutResponse{
		Id:                   "payout_123",
		IdempotencyKey:       "ref_123",
		Amount:               "100.00",
		Currency:             "USD",
		SourceAccountId:      "src_acc",
		DestinationAccountId: "dst_acc",
		Status:               "SUCCEEDED",
		CreatedAt:            now,
	}

	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/payouts/payout_123", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(expectedResp)
	})
	defer server.Close()

	resp, err := c.GetPayoutStatus(context.Background(), "payout_123")
	require.NoError(t, err)
	require.Equal(t, "SUCCEEDED", resp.Status)
}

func TestGetPayoutStatus_HTTPError(t *testing.T) {
	t.Parallel()

	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "not found"}`))
	})
	defer server.Close()

	resp, err := c.GetPayoutStatus(context.Background(), "payout_123")
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "404")
}

func TestGetPayoutStatus_InvalidJSON(t *testing.T) {
	t.Parallel()

	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid`))
	})
	defer server.Close()

	resp, err := c.GetPayoutStatus(context.Background(), "payout_123")
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "unmarshal")
}

// TestCreateTransfer tests the CreateTransfer client method
func TestCreateTransfer_Success(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Format(time.RFC3339)
	expectedResp := TransferResponse{
		Id:                   "transfer_123",
		IdempotencyKey:       "ref_456",
		Amount:               "500.00",
		Currency:             "EUR",
		SourceAccountId:      "src_acc",
		DestinationAccountId: "dst_acc",
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
		err = json.Unmarshal(body, &req)
		require.NoError(t, err)
		require.Equal(t, "ref_456", req.IdempotencyKey)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(expectedResp)
	})
	defer server.Close()

	desc := "Test transfer"
	req := &TransferRequest{
		IdempotencyKey:       "ref_456",
		Amount:               "500.00",
		Currency:             "EUR",
		SourceAccountId:      "src_acc",
		DestinationAccountId: "dst_acc",
		Description:          &desc,
	}

	resp, err := c.CreateTransfer(context.Background(), req)
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

	req := &TransferRequest{IdempotencyKey: "ref_456"}
	resp, err := c.CreateTransfer(context.Background(), req)
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

	req := &TransferRequest{IdempotencyKey: "ref_456"}
	resp, err := c.CreateTransfer(context.Background(), req)
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "unmarshal")
}

// TestGetTransferStatus tests the GetTransferStatus client method
func TestGetTransferStatus_Success(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Format(time.RFC3339)
	expectedResp := TransferResponse{
		Id:                   "transfer_123",
		IdempotencyKey:       "ref_456",
		Amount:               "500.00",
		Currency:             "EUR",
		SourceAccountId:      "src_acc",
		DestinationAccountId: "dst_acc",
		Status:               "SUCCEEDED",
		CreatedAt:            now,
	}

	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/transfers/transfer_123", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(expectedResp)
	})
	defer server.Close()

	resp, err := c.GetTransferStatus(context.Background(), "transfer_123")
	require.NoError(t, err)
	require.Equal(t, "SUCCEEDED", resp.Status)
}

func TestGetTransferStatus_HTTPError(t *testing.T) {
	t.Parallel()

	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "not found"}`))
	})
	defer server.Close()

	resp, err := c.GetTransferStatus(context.Background(), "transfer_123")
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "404")
}

func TestGetTransferStatus_InvalidJSON(t *testing.T) {
	t.Parallel()

	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`broken`))
	})
	defer server.Close()

	resp, err := c.GetTransferStatus(context.Background(), "transfer_123")
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "unmarshal")
}

// TestCreateBankAccount tests the CreateBankAccount client method
func TestCreateBankAccount_Success(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Format(time.RFC3339)
	iban := "FR7630006000011234567890189"
	expectedResp := BankAccountResponse{
		Id:        "ba_123",
		Name:      "Test Account",
		IBAN:      &iban,
		CreatedAt: now,
	}

	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/bank-accounts", r.URL.Path)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer func() { _ = r.Body.Close() }()

		var req BankAccountRequest
		err = json.Unmarshal(body, &req)
		require.NoError(t, err)
		require.Equal(t, "Test Account", req.Name)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(expectedResp)
	})
	defer server.Close()

	req := &BankAccountRequest{
		Name: "Test Account",
		IBAN: &iban,
	}

	resp, err := c.CreateBankAccount(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, expectedResp.Id, resp.Id)
	require.Equal(t, expectedResp.Name, resp.Name)
}

func TestCreateBankAccount_HTTPError(t *testing.T) {
	t.Parallel()

	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "invalid IBAN"}`))
	})
	defer server.Close()

	req := &BankAccountRequest{Name: "Test"}
	resp, err := c.CreateBankAccount(context.Background(), req)
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "400")
}

func TestCreateBankAccount_InvalidJSON(t *testing.T) {
	t.Parallel()

	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{not valid}`))
	})
	defer server.Close()

	req := &BankAccountRequest{Name: "Test"}
	resp, err := c.CreateBankAccount(context.Background(), req)
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "unmarshal")
}

// TestNew tests the New client constructor
func TestNew_CreatesClient(t *testing.T) {
	t.Parallel()

	c := New("test-connector", "api-key-123", "https://api.example.com")
	require.NotNil(t, c)
}

// TestAPITransport tests the authorization header injection
func TestAPITransport_AddsAuthHeader(t *testing.T) {
	t.Parallel()

	var capturedHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	transport := &apiTransport{
		APIKey:     "test-api-key",
		underlying: http.DefaultTransport,
	}

	client := &http.Client{Transport: transport}
	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	_, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, "Bearer test-api-key", capturedHeader)
}

// TestCreatePayout_WithMetadata tests payout creation with metadata
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
			Id:        "payout_meta",
			CreatedAt: now,
			Status:    "PENDING",
			Metadata:  req.Metadata,
		})
	})
	defer server.Close()

	req := &PayoutRequest{
		IdempotencyKey: "ref_meta",
		Metadata:       map[string]string{"key1": "value1"},
	}

	resp, err := c.CreatePayout(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "value1", resp.Metadata["key1"])
}

// TestCreateTransfer_WithDescription tests transfer creation with description
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

		desc := req.Description
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(TransferResponse{
			Id:          "transfer_desc",
			CreatedAt:   now,
			Status:      "PENDING",
			Description: desc,
		})
	})
	defer server.Close()

	desc := "Payment for services"
	req := &TransferRequest{
		IdempotencyKey: "ref_desc",
		Description:    &desc,
	}

	resp, err := c.CreateTransfer(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp.Description)
	require.Equal(t, desc, *resp.Description)
}

// TestCreateBankAccount_AllFields tests bank account creation with all fields
func TestCreateBankAccount_AllFields(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Format(time.RFC3339)
	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		defer func() { _ = r.Body.Close() }()

		var req BankAccountRequest
		_ = json.Unmarshal(body, &req)
		require.Equal(t, "Test Account", req.Name)
		require.NotNil(t, req.AccountNumber)
		require.NotNil(t, req.IBAN)
		require.NotNil(t, req.SwiftBicCode)
		require.NotNil(t, req.Country)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(BankAccountResponse{
			Id:            "ba_full",
			Name:          req.Name,
			AccountNumber: req.AccountNumber,
			IBAN:          req.IBAN,
			SwiftBicCode:  req.SwiftBicCode,
			Country:       req.Country,
			CreatedAt:     now,
			Metadata:      req.Metadata,
		})
	})
	defer server.Close()

	accNum := "123456789"
	iban := "FR7630006000011234567890189"
	swift := "BNPAFRPPXXX"
	country := "FR"

	req := &BankAccountRequest{
		Name:          "Test Account",
		AccountNumber: &accNum,
		IBAN:          &iban,
		SwiftBicCode:  &swift,
		Country:       &country,
		Metadata:      map[string]string{"test": "value"},
	}

	resp, err := c.CreateBankAccount(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "ba_full", resp.Id)
	require.Equal(t, "Test Account", resp.Name)
	require.Equal(t, accNum, *resp.AccountNumber)
	require.Equal(t, iban, *resp.IBAN)
	require.Equal(t, swift, *resp.SwiftBicCode)
	require.Equal(t, country, *resp.Country)
}

// TestGetPayoutStatus_AllStatuses tests different payout statuses
func TestGetPayoutStatus_AllStatuses(t *testing.T) {
	t.Parallel()

	statuses := []string{"PENDING", "SUCCEEDED", "FAILED"}
	for _, status := range statuses {
		status := status
		t.Run(status, func(t *testing.T) {
			t.Parallel()

			now := time.Now().UTC().Format(time.RFC3339)
			c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(PayoutResponse{
					Id:        "payout_status",
					CreatedAt: now,
					Status:    status,
				})
			})
			defer server.Close()

			resp, err := c.GetPayoutStatus(context.Background(), "payout_status")
			require.NoError(t, err)
			require.Equal(t, status, resp.Status)
		})
	}
}

// TestGetTransferStatus_AllStatuses tests different transfer statuses
func TestGetTransferStatus_AllStatuses(t *testing.T) {
	t.Parallel()

	statuses := []string{"PENDING", "SUCCEEDED", "FAILED"}
	for _, status := range statuses {
		status := status
		t.Run(status, func(t *testing.T) {
			t.Parallel()

			now := time.Now().UTC().Format(time.RFC3339)
			c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(TransferResponse{
					Id:        "transfer_status",
					CreatedAt: now,
					Status:    status,
				})
			})
			defer server.Close()

			resp, err := c.GetTransferStatus(context.Background(), "transfer_status")
			require.NoError(t, err)
			require.Equal(t, status, resp.Status)
		})
	}
}

// TestListAccounts tests the ListAccounts method
func TestListAccounts_Success(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.URL.Path, "/accounts")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"id":        "acc_1",
				"name":      "Test Account",
				"currency":  "USD",
				"createdAt": now.Format(time.RFC3339),
			},
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

// TestGetBalances tests the GetBalances method
func TestGetBalances_Success(t *testing.T) {
	t.Parallel()

	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.URL.Path, "/accounts/acc_123/balances")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"balances": []map[string]interface{}{
				{
					"currency": "USD",
					"amount":   "1000",
				},
			},
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

// TestListBeneficiaries tests the ListBeneficiaries method
func TestListBeneficiaries_Success(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.URL.Path, "/beneficiaries")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"id":        "ben_1",
				"name":      "Test Beneficiary",
				"createdAt": now.Format(time.RFC3339),
			},
		})
	})
	defer server.Close()

	beneficiaries, err := c.ListBeneficiaries(context.Background(), 1, 10, time.Time{})
	require.NoError(t, err)
	require.Len(t, beneficiaries, 1)
	require.Equal(t, "ben_1", beneficiaries[0].Id)
}

func TestListBeneficiaries_WithCreatedAtFrom(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		require.Contains(t, r.URL.RawQuery, "createdAtFrom")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
	})
	defer server.Close()

	beneficiaries, err := c.ListBeneficiaries(context.Background(), 1, 10, now.Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, beneficiaries, 0)
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

// TestListTransactions tests the ListTransactions method
func TestListTransactions_Success(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.URL.Path, "/transactions")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"id":        "tx_1",
				"amount":    "1000",
				"currency":  "USD",
				"type":      "PAYIN",
				"status":    "SUCCEEDED",
				"createdAt": now.Format(time.RFC3339),
				"updatedAt": now.Format(time.RFC3339),
			},
		})
	})
	defer server.Close()

	transactions, err := c.ListTransactions(context.Background(), 1, 10, time.Time{})
	require.NoError(t, err)
	require.Len(t, transactions, 1)
	require.Equal(t, "tx_1", transactions[0].Id)
}

func TestListTransactions_WithUpdatedAtFrom(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	c, server := mockAPIClient(func(w http.ResponseWriter, r *http.Request) {
		require.Contains(t, r.URL.RawQuery, "updatedAtFrom")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
	})
	defer server.Close()

	transactions, err := c.ListTransactions(context.Background(), 1, 10, now.Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, transactions, 0)
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

func TestCreatePayout_ReadBodyError(t *testing.T) {
	t.Parallel()

	// Create a custom transport that returns an error reader
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Server writes headers but provides a body that will fail to read
		w.WriteHeader(http.StatusOK)
		// Sending partial/invalid chunked response is hard to simulate,
		// so we test by ensuring error handling works for various server errors
	}))
	defer server.Close()

	configuration := genericclient.NewConfiguration()
	configuration.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	configuration.Servers[0].URL = server.URL

	c := &client{apiClient: genericclient.NewAPIClient(configuration)}

	req := &PayoutRequest{IdempotencyKey: "ref_123", Amount: "100.00", Currency: "USD"}
	// This should succeed but return empty body, which will fail unmarshal
	_, err := c.CreatePayout(context.Background(), req)
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

	req := &TransferRequest{IdempotencyKey: "ref_123", Amount: "100.00", Currency: "USD"}
	_, err := c.CreateTransfer(context.Background(), req)
	require.Error(t, err)
}

func TestCreateBankAccount_ReadBodyError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	configuration := genericclient.NewConfiguration()
	configuration.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	configuration.Servers[0].URL = server.URL

	c := &client{apiClient: genericclient.NewAPIClient(configuration)}

	req := &BankAccountRequest{Name: "Test"}
	_, err := c.CreateBankAccount(context.Background(), req)
	require.Error(t, err)
}

func TestGetPayoutStatus_ReadBodyError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	configuration := genericclient.NewConfiguration()
	configuration.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	configuration.Servers[0].URL = server.URL

	c := &client{apiClient: genericclient.NewAPIClient(configuration)}

	_, err := c.GetPayoutStatus(context.Background(), "payout_123")
	require.Error(t, err)
}

func TestGetTransferStatus_ReadBodyError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	configuration := genericclient.NewConfiguration()
	configuration.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	configuration.Servers[0].URL = server.URL

	c := &client{apiClient: genericclient.NewAPIClient(configuration)}

	_, err := c.GetTransferStatus(context.Background(), "transfer_123")
	require.Error(t, err)
}
