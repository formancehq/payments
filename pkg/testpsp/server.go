package testpsp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"time"
)

type AccountData struct {
	ID          string    `json:"id"`
	AccountName string    `json:"accountName"`
	CreatedAt   time.Time `json:"createdAt"`
}

type BalanceEntry struct {
	Currency string `json:"currency"`
	Amount   string `json:"amount"`
}

type BalancesData struct {
	ID        string         `json:"id"`
	AccountID string         `json:"accountID"`
	At        time.Time      `json:"at"`
	Balances  []BalanceEntry `json:"balances"`
}

type BeneficiaryData struct {
	ID        string    `json:"id"`
	OwnerName string    `json:"ownerName"`
	CreatedAt time.Time `json:"createdAt"`
}

type TransactionData struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Currency  string    `json:"currency"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	Amount    string    `json:"amount"`

	SourceAccountID      *string `json:"sourceAccountID,omitempty"`
	DestinationAccountID *string `json:"destinationAccountID,omitempty"`
}

type PayoutRequest struct {
	IdempotencyKey       string            `json:"idempotencyKey"`
	Amount               string            `json:"amount"`
	Currency             string            `json:"currency"`
	SourceAccountID      string            `json:"sourceAccountId"`
	DestinationAccountID string            `json:"destinationAccountId"`
	Description          *string           `json:"description,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
}

type PayoutResponse struct {
	ID                   string `json:"id"`
	IdempotencyKey       string `json:"idempotencyKey"`
	Amount               string `json:"amount"`
	Currency             string `json:"currency"`
	SourceAccountID      string `json:"sourceAccountId"`
	DestinationAccountID string `json:"destinationAccountId"`
	Status               string `json:"status"`
	CreatedAt            string `json:"createdAt"`
}

type TransferRequest struct {
	IdempotencyKey       string            `json:"idempotencyKey"`
	Amount               string            `json:"amount"`
	Currency             string            `json:"currency"`
	SourceAccountID      string            `json:"sourceAccountId"`
	DestinationAccountID string            `json:"destinationAccountId"`
	Description          *string           `json:"description,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
}

type TransferResponse struct {
	ID                   string `json:"id"`
	IdempotencyKey       string `json:"idempotencyKey"`
	Amount               string `json:"amount"`
	Currency             string `json:"currency"`
	SourceAccountID      string `json:"sourceAccountId"`
	DestinationAccountID string `json:"destinationAccountId"`
	Status               string `json:"status"`
	CreatedAt            string `json:"createdAt"`
}

type Server struct {
	httpServer    *httptest.Server
	Accounts      []AccountData
	Beneficiaries []BeneficiaryData
	Transactions  []TransactionData
	balances      map[string]BalancesData

	accountsCalled      atomic.Int64
	balancesCalled      atomic.Int64
	transactionsCalled  atomic.Int64
	beneficiariesCalled atomic.Int64
	payoutsCalled       atomic.Int64
	transfersCalled     atomic.Int64

	lastAccountCreatedAtFromNano     atomic.Int64
	lastBeneficiaryCreatedAtFromNano atomic.Int64
	lastTransactionUpdatedAtFromNano atomic.Int64
}

func NewServer() *Server {
	now := time.Now().UTC().Truncate(time.Second)
	src := "acc-001"

	s := &Server{
		Accounts: []AccountData{
			{ID: "acc-001", AccountName: "Test Account One", CreatedAt: now.Add(-2 * time.Hour)},
			{ID: "acc-002", AccountName: "Test Account Two", CreatedAt: now.Add(-1 * time.Hour)},
		},
		Beneficiaries: []BeneficiaryData{
			{ID: "ben-001", OwnerName: "Test Beneficiary One", CreatedAt: now.Add(-45 * time.Minute)},
		},
		Transactions: []TransactionData{
			{
				ID:        "tx-001",
				CreatedAt: now.Add(-30 * time.Minute),
				UpdatedAt: now.Add(-25 * time.Minute),
				Currency:  "USD/2",
				Type:      "PAYIN",
				Status:    "SUCCEEDED",
				Amount:    "5000",
			},
			{
				ID:                   "tx-002",
				CreatedAt:            now.Add(-15 * time.Minute),
				UpdatedAt:            now.Add(-10 * time.Minute),
				Currency:             "USD/2",
				Type:                 "PAYOUT",
				Status:               "SUCCEEDED",
				Amount:               "2000",
				SourceAccountID:      &src,
			},
		},
		balances: map[string]BalancesData{
			"acc-001": {ID: "bal-001", AccountID: "acc-001", At: now, Balances: []BalanceEntry{{Currency: "USD/2", Amount: "10000"}}},
			"acc-002": {ID: "bal-002", AccountID: "acc-002", At: now, Balances: []BalanceEntry{{Currency: "USD/2", Amount: "25000"}}},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/accounts/", s.handleAccountSub)
	mux.HandleFunc("/accounts", s.handleAccounts)
	mux.HandleFunc("/transactions", s.handleTransactions)
	mux.HandleFunc("/beneficiaries", s.handleBeneficiaries)
	mux.HandleFunc("/payouts", s.handlePayouts)
	mux.HandleFunc("/transfers", s.handleTransfers)

	s.httpServer = httptest.NewServer(mux)
	return s
}

func (s *Server) URL() string                { return s.httpServer.URL }
func (s *Server) Close()                     { s.httpServer.Close() }
func (s *Server) AccountsCalled() int64      { return s.accountsCalled.Load() }
func (s *Server) BalancesCalled() int64      { return s.balancesCalled.Load() }
func (s *Server) TransactionsCalled() int64  { return s.transactionsCalled.Load() }
func (s *Server) BeneficiariesCalled() int64 { return s.beneficiariesCalled.Load() }
func (s *Server) PayoutsCalled() int64       { return s.payoutsCalled.Load() }
func (s *Server) TransfersCalled() int64     { return s.transfersCalled.Load() }

func (s *Server) LastSeenAccountPagingParamCreatedAtFrom() time.Time {
	return time.Unix(0, s.lastAccountCreatedAtFromNano.Load()).UTC()
}

func (s *Server) LastSeenBeneficiaryPagingParamCreatedAtFrom() time.Time {
	return time.Unix(0, s.lastBeneficiaryCreatedAtFromNano.Load()).UTC()
}

func (s *Server) LastSeenTransactionPagingParamUpdatedAtFrom() time.Time {
	return time.Unix(0, s.lastTransactionUpdatedAtFromNano.Load()).UTC()
}

func parseTimeParam(r *http.Request, name string) (time.Time, bool) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339Nano, v)
	if err != nil {
		t, err = time.Parse(time.RFC3339, v)
		if err != nil {
			return time.Time{}, false
		}
	}
	return t, true
}

func (s *Server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	s.accountsCalled.Add(1)
	w.Header().Set("Content-Type", "application/json")
	if t, ok := parseTimeParam(r, "createdAtFrom"); ok {
		s.lastAccountCreatedAtFromNano.Store(t.UnixNano())
		_ = json.NewEncoder(w).Encode([]AccountData{})
		return
	}
	_ = json.NewEncoder(w).Encode(s.Accounts)
}

func (s *Server) handleAccountSub(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/accounts/"), "/")
	if len(parts) >= 2 && parts[1] == "balances" {
		accountID := parts[0]
		b, ok := s.balances[accountID]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		s.balancesCalled.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(b)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func (s *Server) handleTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	s.transactionsCalled.Add(1)
	w.Header().Set("Content-Type", "application/json")
	if t, ok := parseTimeParam(r, "updatedAtFrom"); ok {
		s.lastTransactionUpdatedAtFromNano.Store(t.UnixNano())
		_ = json.NewEncoder(w).Encode([]TransactionData{})
		return
	}
	_ = json.NewEncoder(w).Encode(s.Transactions)
}

func (s *Server) handleBeneficiaries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	s.beneficiariesCalled.Add(1)
	w.Header().Set("Content-Type", "application/json")
	if t, ok := parseTimeParam(r, "createdAtFrom"); ok {
		s.lastBeneficiaryCreatedAtFromNano.Store(t.UnixNano())
		_ = json.NewEncoder(w).Encode([]BeneficiaryData{})
		return
	}
	_ = json.NewEncoder(w).Encode(s.Beneficiaries)
}

func (s *Server) handlePayouts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req PayoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if req.IdempotencyKey == "" || req.Amount == "" || req.Currency == "" || req.SourceAccountID == "" || req.DestinationAccountID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	s.payoutsCalled.Add(1)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(PayoutResponse{
		ID:                   "payout-001",
		IdempotencyKey:       req.IdempotencyKey,
		Amount:               req.Amount,
		Currency:             req.Currency,
		SourceAccountID:      req.SourceAccountID,
		DestinationAccountID: req.DestinationAccountID,
		Status:               "SUCCEEDED",
		CreatedAt:            time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleTransfers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if req.IdempotencyKey == "" || req.Amount == "" || req.Currency == "" || req.SourceAccountID == "" || req.DestinationAccountID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	s.transfersCalled.Add(1)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(TransferResponse{
		ID:                   "transfer-001",
		IdempotencyKey:       req.IdempotencyKey,
		Amount:               req.Amount,
		Currency:             req.Currency,
		SourceAccountID:      req.SourceAccountID,
		DestinationAccountID: req.DestinationAccountID,
		Status:               "SUCCEEDED",
		CreatedAt:            time.Now().UTC().Format(time.RFC3339),
	})
}
