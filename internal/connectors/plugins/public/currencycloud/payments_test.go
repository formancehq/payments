package currencycloud

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/types/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/currencycloud/client"
	"github.com/formancehq/payments/pkg/domain/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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
				State: []byte(fmt.Sprintf(
					`{"lastUpdatedAt": "%s", "lastProcessedID": "%s"}`,
					sampleTransactions[38].UpdatedAt.Format(time.RFC3339Nano),
					sampleTransactions[38].ID,
				)),
				PageSize: 40,
			}

			// Both pages query with the STABLE oldState watermark (no mid-pagination mutation).
			m.EXPECT().GetTransactions(gomock.Any(), 1, 40, sampleTransactions[38].UpdatedAt.UTC()).Return(
				sampleTransactions[:40],
				2,
				nil,
			)

			m.EXPECT().GetTransactions(gomock.Any(), 2, 40, sampleTransactions[38].UpdatedAt.UTC()).Return(
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

		It("keeps distinct transactions that share the watermark timestamp (M-CON2)", func(ctx SpecContext) {
			ts := now.Add(-time.Hour).UTC()
			mk := func(id string) client.Transaction {
				return client.Transaction{
					ID:        id,
					AccountID: "acc",
					Currency:  "EUR",
					Type:      "credit",
					Status:    "completed",
					CreatedAt: ts,
					UpdatedAt: ts,
					Amount:    "100",
				}
			}

			req := models.FetchNextPaymentsRequest{
				State:    []byte(fmt.Sprintf(`{"lastUpdatedAt": "%s", "lastProcessedID": "a"}`, ts.Format(time.RFC3339Nano))),
				PageSize: 40,
			}
			m.EXPECT().GetTransactions(gomock.Any(), 1, 40, ts.UTC()).Return(
				[]client.Transaction{mk("a"), mk("b"), mk("c")},
				-1,
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			// "a" was the already-processed boundary row; "b" and "c" share its
			// timestamp and must NOT be dropped.
			Expect(resp.Payments).To(HaveLen(2))
			Expect([]string{resp.Payments[0].Reference, resp.Payments[1].Reference}).To(ConsistOf("b", "c"))
		})

		It("walks a same-second group larger than PageSize across cycles without stalling", func(ctx SpecContext) {
			ts := now.Add(-time.Hour).UTC()
			mk := func(id string) client.Transaction {
				return client.Transaction{
					ID:        id,
					AccountID: "acc",
					Currency:  "EUR",
					Type:      "credit",
					Status:    "completed",
					CreatedAt: ts,
					UpdatedAt: ts,
					Amount:    "100",
				}
			}
			all := []client.Transaction{mk("t0"), mk("t1"), mk("t2"), mk("t3"), mk("t4")}
			refs := func(ps []models.PSPPayment) []string {
				out := make([]string, len(ps))
				for i := range ps {
					out[i] = ps[i].Reference
				}
				return out
			}

			// Cycle 1: page 1 -> t0, t1.
			m.EXPECT().GetTransactions(gomock.Any(), 1, 2, time.Time{}).Return(all[0:2], 2, nil)
			resp, err := plg.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{State: []byte(`{}`), PageSize: 2})
			Expect(err).To(BeNil())
			Expect(refs(resp.Payments)).To(Equal([]string{"t0", "t1"}))

			// Cycle 2: page 1 re-fetched (t1 deduped) then page 2 -> t2, t3.
			m.EXPECT().GetTransactions(gomock.Any(), 1, 2, ts.UTC()).Return(all[0:2], 2, nil)
			m.EXPECT().GetTransactions(gomock.Any(), 2, 2, ts.UTC()).Return(all[2:4], 3, nil)
			resp, err = plg.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{State: resp.NewState, PageSize: 2})
			Expect(err).To(BeNil())
			// Boundary t1 is deduped; t0 (a same-second sibling on the re-fetched
			// page 1) is re-emitted by design — storage upserts dedup it. The exact
			// assertion catches any unintended extra re-emission.
			Expect(refs(resp.Payments)).To(Equal([]string{"t0", "t2", "t3"}))

			// Cycle 3: page 3 -> t4 (group fully drained on a short final page).
			m.EXPECT().GetTransactions(gomock.Any(), 3, 2, ts.UTC()).Return(all[4:5], -1, nil)
			resp, err = plg.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{State: resp.NewState, PageSize: 2})
			Expect(err).To(BeNil())
			Expect(refs(resp.Payments)).To(Equal([]string{"t4"}))

			// Cycle 4: a newer-second transaction t5 lands on the short last page
			// (3). The cursor must stay on page 3 rather than advance to page 4, or
			// t5 would be stranded forever behind an empty page.
			ts2 := ts.Add(time.Second)
			t5 := client.Transaction{
				ID:        "t5",
				AccountID: "acc",
				Currency:  "EUR",
				Type:      "credit",
				Status:    "completed",
				CreatedAt: ts2,
				UpdatedAt: ts2,
				Amount:    "100",
			}
			m.EXPECT().GetTransactions(gomock.Any(), 3, 2, ts.UTC()).Return([]client.Transaction{all[4], t5}, -1, nil)
			resp, err = plg.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{State: resp.NewState, PageSize: 2})
			Expect(err).To(BeNil())
			Expect(refs(resp.Payments)).To(Equal([]string{"t5"}))
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
