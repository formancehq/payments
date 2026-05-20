package mappers

import (
	"strings"
	"testing"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/internal/models"
)

func TestWithdrawalRequestToPSPPayment_FullEnumCoverage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		in         client.WithdrawalRequest
		wantStatus models.PaymentStatus
		wantScheme models.PaymentScheme
	}{
		{
			name:       "SEPA pending",
			in:         client.WithdrawalRequest{ID: 1, Datetime: "2025-09-25 14:42:59", Type: 0, Currency: "EUR", Amount: "100.00", Status: 0},
			wantStatus: models.PAYMENT_STATUS_PENDING,
			wantScheme: models.PAYMENT_SCHEME_SEPA_CREDIT,
		},
		{
			name:       "SEPA in-progress is also PENDING",
			in:         client.WithdrawalRequest{ID: 2, Datetime: "2025-09-25 14:42:59", Type: 0, Currency: "EUR", Amount: "100.00", Status: 1},
			wantStatus: models.PAYMENT_STATUS_PENDING,
			wantScheme: models.PAYMENT_SCHEME_SEPA_CREDIT,
		},
		{
			name:       "international wire finished",
			in:         client.WithdrawalRequest{ID: 3, Datetime: "2025-09-25 14:42:59", Type: 1, Currency: "USD", Amount: "100.00", Status: 2},
			wantStatus: models.PAYMENT_STATUS_SUCCEEDED,
			wantScheme: models.PAYMENT_SCHEME_OTHER,
		},
		{
			name:       "ARDI canceled",
			in:         client.WithdrawalRequest{ID: 4, Datetime: "2025-09-25 14:42:59", Type: 2, Currency: "USD", Amount: "100.00", Status: 3},
			wantStatus: models.PAYMENT_STATUS_CANCELLED,
			wantScheme: models.PAYMENT_SCHEME_OTHER,
		},
		{
			name:       "international BIC failed",
			in:         client.WithdrawalRequest{ID: 5, Datetime: "2025-09-25 14:42:59", Type: 3, Currency: "USD", Amount: "100.00", Status: 4},
			wantStatus: models.PAYMENT_STATUS_FAILED,
			wantScheme: models.PAYMENT_SCHEME_OTHER,
		},
		{
			name:       "crypto withdrawal finished",
			in:         client.WithdrawalRequest{ID: 6, Datetime: "2025-09-25 14:42:59", Type: 4, Currency: "BTC", Amount: "0.001", Status: 2, Network: "bitcoin", Address: "3FiK", TxID: "wd-1"},
			wantStatus: models.PAYMENT_STATUS_SUCCEEDED,
			wantScheme: models.PAYMENT_SCHEME_OTHER,
		},
		{
			name:       "unknown type and status fall back to UNKNOWN",
			in:         client.WithdrawalRequest{ID: 7, Datetime: "2025-09-25 14:42:59", Type: 99, Currency: "EUR", Amount: "1.00", Status: 99},
			wantStatus: models.PAYMENT_STATUS_UNKNOWN,
			wantScheme: models.PAYMENT_SCHEME_UNKNOWN,
		},
	}

	cur := map[string]int{"EUR": 2, "USD": 2, "BTC": 8}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := WithdrawalRequestToPSPPayment(cur, tc.in)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if got == nil {
				t.Fatal("expected non-nil PSPPayment")
			}
			if got.Type != models.PAYMENT_TYPE_PAYOUT {
				t.Errorf("Type = %v, want PAYOUT", got.Type)
			}
			if got.Status != tc.wantStatus {
				t.Errorf("Status = %v, want %v", got.Status, tc.wantStatus)
			}
			if got.Scheme != tc.wantScheme {
				t.Errorf("Scheme = %v, want %v", got.Scheme, tc.wantScheme)
			}
			if !strings.HasPrefix(got.Reference, "wr:") {
				t.Errorf("Reference must use wr: prefix, got %q", got.Reference)
			}
			if got.Metadata[MetadataKeySource] != PaymentSourceWithdrawalRequests {
				t.Errorf("Source metadata wrong: %q", got.Metadata[MetadataKeySource])
			}
		})
	}
}

func TestWithdrawalRequestToPSPPayment_UnknownCurrencyReturnsNil(t *testing.T) {
	t.Parallel()
	got, err := WithdrawalRequestToPSPPayment(
		map[string]int{"EUR": 2},
		client.WithdrawalRequest{ID: 1, Datetime: "2025-09-25 14:42:59", Type: 4, Currency: "FUTURE_COIN", Amount: "1", Status: 2},
	)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != nil {
		t.Errorf("unknown currency must return nil, got %+v", got)
	}
}

func TestWithdrawalRequestToPSPPayment_MissingIDIsError(t *testing.T) {
	t.Parallel()
	_, err := WithdrawalRequestToPSPPayment(map[string]int{"EUR": 2}, client.WithdrawalRequest{Datetime: "2025-09-25 14:42:59", Currency: "EUR", Amount: "1", Status: 2})
	if err == nil {
		t.Error("expected error on missing id")
	}
}

func TestWithdrawalRequestToPSPPayment_BadDatetimeIsError(t *testing.T) {
	t.Parallel()
	_, err := WithdrawalRequestToPSPPayment(map[string]int{"EUR": 2},
		client.WithdrawalRequest{ID: 1, Datetime: "bad", Currency: "EUR", Amount: "1", Status: 2})
	if err == nil {
		t.Error("expected error on unparseable datetime")
	}
}

func TestWithdrawalRequestToPSPPayment_BadAmountIsError(t *testing.T) {
	t.Parallel()
	_, err := WithdrawalRequestToPSPPayment(map[string]int{"EUR": 2},
		client.WithdrawalRequest{ID: 1, Datetime: "2025-09-25 14:42:59", Currency: "EUR", Amount: "not-a-number", Status: 2})
	if err == nil {
		t.Error("expected error on bad amount")
	}
}
