package currencycloud

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/currencycloud/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	"github.com/golang/mock/gomock"
)

var _ = Describe("CurrencyCloud Plugin Payments", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  models.Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetching next payments", func() {
		var (
			sampleTransactions []client.Transaction
			now                time.Time
		)

		BeforeEach(func() {
			now = time.Now().UTC()

			sampleTransactions = make([]client.Transaction, 0)
			for i := 0; i < 50; i++ {
				sampleTransactions = append(sampleTransactions, client.Transaction{
					ID:        fmt.Sprintf("%d", i),
					AccountID: fmt.Sprintf("Account-%d", i),
					Currency:  "EUR",
					Type:      "credit",
					Status:    "completed",
					CreatedAt: now.Add(-time.Duration(60-i) * time.Minute).UTC(),
					UpdatedAt: now.Add(-time.Duration(60-i-1) * time.Minute).UTC(),
					Amount:    "100",
				})
			}
		})

		It("should return an error - get transactions error", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetTransactions(gomock.Any(), 1, 60, time.Time{}).Return(
				[]client.Transaction{},
				-1,
				errors.New("test error"),
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should fetch next payments - no state no results", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetTransactions(gomock.Any(), 1, 60, time.Time{}).Return(
				[]client.Transaction{},
				-1,
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.LastUpdatedAt.IsZero()).To(BeTrue())
		})

		It("should fetch next payments - no state pageSize > total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetTransactions(gomock.Any(), 1, 60, time.Time{}).Return(
				sampleTransactions,
				-1,
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(50))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.LastUpdatedAt).To(Equal(sampleTransactions[49].UpdatedAt))
		})

		It("should fetch next payments - no state pageSize < total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 40,
			}

			m.EXPECT().GetTransactions(gomock.Any(), 1, 40, time.Time{}).Return(
				sampleTransactions[:40],
				2,
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(40))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastUpdatedAt).To(Equal(sampleTransactions[39].UpdatedAt))
		})

		It("should fetch next payments - with state pageSize < total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(fmt.Sprintf(`{"lastUpdatedAt": "%s"}`, sampleTransactions[38].UpdatedAt.Format(time.RFC3339Nano))),
				PageSize: 40,
			}

			m.EXPECT().GetTransactions(gomock.Any(), 1, 40, sampleTransactions[38].UpdatedAt.UTC()).Return(
				sampleTransactions[:40],
				2,
				nil,
			)

			m.EXPECT().GetTransactions(gomock.Any(), 2, 40, sampleTransactions[39].UpdatedAt.UTC()).Return(
				sampleTransactions[41:],
				-1,
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(10))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.LastUpdatedAt).To(Equal(sampleTransactions[49].UpdatedAt))
		})
	})
})

func TestTransactionToPayment(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	t.Run("unsupported currencies", func(t *testing.T) {
		t.Parallel()

		transaction := client.Transaction{
			ID:        "test",
			AccountID: "test",
			Currency:  "HUF",
			Type:      "credit",
			Status:    "completed",
			CreatedAt: now,
			UpdatedAt: now,
			Amount:    "100",
		}

		p, err := transactionToPayment(transaction)
		require.NoError(t, err)
		require.Nil(t, p)
	})

	t.Run("wrong amount string", func(t *testing.T) {
		t.Parallel()

		transaction := client.Transaction{
			ID:        "test",
			AccountID: "test",
			Currency:  "EUR",
			Type:      "credit",
			Status:    "completed",
			CreatedAt: now,
			UpdatedAt: now,
			Amount:    "100,fdv",
		}

		p, err := transactionToPayment(transaction)
		require.Error(t, err)
		require.Nil(t, p)
	})

	t.Run("credit payment type - status completed", func(t *testing.T) {
		t.Parallel()

		transaction := client.Transaction{
			ID:        "test",
			AccountID: "test",
			Currency:  "EUR",
			Type:      "credit",
			Status:    "completed",
			CreatedAt: now,
			UpdatedAt: now,
			Amount:    "100",
		}

		p, err := transactionToPayment(transaction)
		require.NoError(t, err)
		require.NotNil(t, p)

		expected := models.PSPPayment{
			ParentReference:             "",
			Reference:                   transaction.ID,
			CreatedAt:                   now,
			Type:                        models.PAYMENT_TYPE_PAYIN,
			Amount:                      big.NewInt(10000),
			Asset:                       "EUR/2",
			Scheme:                      models.PAYMENT_SCHEME_OTHER,
			Status:                      models.PAYMENT_STATUS_SUCCEEDED,
			DestinationAccountReference: pointer.For("test"),
		}

		comparePSPPayments(t, expected, *p)
	})

	t.Run("debit payment type - status pending", func(t *testing.T) {
		t.Parallel()

		transaction := client.Transaction{
			ID:        "test",
			AccountID: "test",
			Currency:  "EUR",
			Type:      "debit",
			Status:    "pending",
			CreatedAt: now,
			UpdatedAt: now,
			Amount:    "100",
		}

		p, err := transactionToPayment(transaction)
		require.NoError(t, err)
		require.NotNil(t, p)

		expected := models.PSPPayment{
			ParentReference:        "",
			Reference:              transaction.ID,
			CreatedAt:              now,
			Type:                   models.PAYMENT_TYPE_PAYOUT,
			Amount:                 big.NewInt(10000),
			Asset:                  "EUR/2",
			Scheme:                 models.PAYMENT_SCHEME_OTHER,
			Status:                 models.PAYMENT_STATUS_PENDING,
			SourceAccountReference: pointer.For("test"),
		}

		comparePSPPayments(t, expected, *p)
	})

	t.Run("other payment type - deleted status", func(t *testing.T) {
		t.Parallel()

		transaction := client.Transaction{
			ID:        "test",
			AccountID: "test",
			Currency:  "EUR",
			Type:      "unknown",
			Status:    "deleted",
			CreatedAt: now,
			UpdatedAt: now,
			Amount:    "100",
		}

		p, err := transactionToPayment(transaction)
		require.NoError(t, err)
		require.NotNil(t, p)

		expected := models.PSPPayment{
			ParentReference: "",
			Reference:       transaction.ID,
			CreatedAt:       now,
			Type:            models.PAYMENT_TYPE_OTHER,
			Amount:          big.NewInt(10000),
			Asset:           "EUR/2",
			Scheme:          models.PAYMENT_SCHEME_OTHER,
			Status:          models.PAYMENT_STATUS_FAILED,
		}

		comparePSPPayments(t, expected, *p)
	})

	t.Run("credit payment type - status unknown", func(t *testing.T) {
		t.Parallel()

		transaction := client.Transaction{
			ID:        "test",
			AccountID: "test",
			Currency:  "EUR",
			Type:      "credit",
			Status:    "unknown",
			CreatedAt: now,
			UpdatedAt: now,
			Amount:    "100",
		}

		p, err := transactionToPayment(transaction)
		require.NoError(t, err)
		require.NotNil(t, p)

		expected := models.PSPPayment{
			ParentReference:             "",
			Reference:                   transaction.ID,
			CreatedAt:                   now,
			Type:                        models.PAYMENT_TYPE_PAYIN,
			Amount:                      big.NewInt(10000),
			Asset:                       "EUR/2",
			Scheme:                      models.PAYMENT_SCHEME_OTHER,
			Status:                      models.PAYMENT_STATUS_OTHER,
			DestinationAccountReference: pointer.For("test"),
		}

		comparePSPPayments(t, expected, *p)
	})
}

func comparePSPPayments(t *testing.T, a, b models.PSPPayment) {
	require.Equal(t, a.ParentReference, b.ParentReference)
	require.Equal(t, a.Reference, b.Reference)
	require.Equal(t, a.CreatedAt, b.CreatedAt)
	require.Equal(t, a.Type, b.Type)
	require.Equal(t, a.Amount, b.Amount)
	require.Equal(t, a.Asset, b.Asset)
	require.Equal(t, a.Scheme, b.Scheme)
	require.Equal(t, a.Status, b.Status)

	switch {
	case a.SourceAccountReference != nil && b.SourceAccountReference != nil:
		require.Equal(t, *a.SourceAccountReference, *b.SourceAccountReference)
	case a.SourceAccountReference == nil && b.SourceAccountReference == nil:
	default:
		t.Fatalf("SourceAccountReference mismatch: %v != %v", a.SourceAccountReference, b.SourceAccountReference)
	}

	switch {
	case a.DestinationAccountReference != nil && b.DestinationAccountReference != nil:
		require.Equal(t, *a.DestinationAccountReference, *b.DestinationAccountReference)
	case a.DestinationAccountReference == nil && b.DestinationAccountReference == nil:
	default:
		t.Fatalf("DestinationAccountReference mismatch: %v != %v", a.DestinationAccountReference, b.DestinationAccountReference)
	}

	require.Equal(t, len(a.Metadata), len(b.Metadata))
	for k, v := range a.Metadata {
		require.Equal(t, v, b.Metadata[k])
	}
}

func TestMatchTransactionType(t *testing.T) {
	tests := []struct {
		entityType          string
		transactionType     string
		expectedPaymentType models.PaymentType
	}{
		{
			entityType:          "inbound_funds",
			transactionType:     "credit",
			expectedPaymentType: models.PAYMENT_TYPE_PAYIN,
		},
		{
			entityType:          "payment",
			transactionType:     "debit",
			expectedPaymentType: models.PAYMENT_TYPE_PAYOUT,
		},
		{
			entityType:          "transfer",
			transactionType:     "debit",
			expectedPaymentType: models.PAYMENT_TYPE_TRANSFER,
		},
		{
			entityType:          "balance_transfer",
			transactionType:     "debit",
			expectedPaymentType: models.PAYMENT_TYPE_TRANSFER,
		},
		{
			entityType:          "unknown",
			transactionType:     "unknown",
			expectedPaymentType: models.PAYMENT_TYPE_OTHER,
		},
		{
			entityType:          "unknown",
			transactionType:     "credit",
			expectedPaymentType: models.PAYMENT_TYPE_PAYIN,
		},
		{
			entityType:          "unknown",
			transactionType:     "debit",
			expectedPaymentType: models.PAYMENT_TYPE_PAYOUT,
		},
	}

	for _, test := range tests {
		t.Run(test.entityType+"-"+test.transactionType, func(t *testing.T) {
			t.Parallel()

			paymentType := matchTransactionType(test.entityType, test.transactionType)
			require.Equal(t, test.expectedPaymentType, paymentType)
		})
	}
}
