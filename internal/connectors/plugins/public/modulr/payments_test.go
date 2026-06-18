package modulr

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/types/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/modulr/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Modulr Plugin Payments", func() {
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

			// Modulr returns transactions newest-first (descending by transactionDate),
			// so index 0 is the most recent. PostedDate intentionally lags TransactionDate
			// by 2h: the watermark must track TransactionDate (the filtered/compared field),
			// never the later PostedDate.
			sampleTransactions = make([]client.Transaction, 0)
			for i := 0; i < 50; i++ {
				txnDate := now.Add(-time.Duration(i) * time.Minute)
				sampleTransactions = append(sampleTransactions, client.Transaction{
					ID:              fmt.Sprintf("%d", i),
					Type:            "PI_FAST",
					Amount:          "100.01",
					Credit:          false,
					SourceID:        fmt.Sprintf("test-%d", i),
					Description:     fmt.Sprintf("Description %d", i),
					PostedDate:      txnDate.Add(2 * time.Hour).Format(transactionDateLayout),
					TransactionDate: txnDate.Format(transactionDateLayout),
					Account: client.Account{
						Currency: "USD",
					},
				})
			}
		})

		It("should return an error - missing from payload", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("missing from payload in request"))
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should return an error - get transactions error", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:       []byte(`{}`),
				PageSize:    60,
				FromPayload: []byte(`{"reference": "test"}`),
			}

			m.EXPECT().GetTransactions(gomock.Any(), "test", 0, 60, timeEq(time.Time{}), timeEq(time.Time{})).Return(
				[]client.Transaction{},
				0,
				errors.New("test error"),
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should fetch next payments - no state no results", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:       []byte(`{}`),
				PageSize:    60,
				FromPayload: []byte(`{"reference": "test"}`),
			}

			m.EXPECT().GetTransactions(gomock.Any(), "test", 0, 60, timeEq(time.Time{}), timeEq(time.Time{})).Return(
				[]client.Transaction{},
				0,
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
			// Nothing newer than the watermark: stay put, no drain in progress.
			Expect(state.LastTransactionTime.IsZero()).To(BeTrue())
			Expect(state.Ceiling.IsZero()).To(BeTrue())
			Expect(state.Version).To(Equal(paymentsStateVersion))
		})

		It("should fetch a single page oldest-first and advance the watermark to the newest transactionDate", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:       []byte(`{}`),
				PageSize:    60,
				FromPayload: []byte(`{"reference": "test"}`),
			}

			// Single page (totalPages = 1): it is both the newest and oldest page.
			m.EXPECT().GetTransactions(gomock.Any(), "test", 0, 60, timeEq(time.Time{}), timeEq(time.Time{})).Return(
				sampleTransactions,
				1,
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(50))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			// Emitted oldest-first (ascending CreatedAt) so the earliest event seeds each
			// payment's base row in the engine.
			expectOldestFirst(resp.Payments)

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.Ceiling.IsZero()).To(BeTrue())
			Expect(state.Version).To(Equal(paymentsStateVersion))

			// Watermark is the newest TransactionDate (index 0), not the PostedDate.
			newest, _ := time.Parse(transactionDateLayout, sampleTransactions[0].TransactionDate)
			Expect(state.LastTransactionTime.UTC()).To(Equal(newest.UTC()))
			postedNewest, _ := time.Parse(transactionDateLayout, sampleTransactions[0].PostedDate)
			Expect(state.LastTransactionTime.UTC()).ToNot(Equal(postedNewest.UTC()))
		})

		It("should drain a multi-page window oldest-first without losing transactions", func(ctx SpecContext) {
			// 50 transactions, PageSize 20 -> 3 pages. Modulr is newest-first, so
			// page 0 = indices 0..19 (newest), page 1 = 20..39, page 2 = 40..49 (oldest).
			// They must be emitted oldest-first: page 2, then page 1, then page 0.
			const pageSize = 20
			ceiling, _ := time.Parse(transactionDateLayout, sampleTransactions[0].TransactionDate)

			// open: peek page 0 (unbounded above) -> freeze ceiling, TotalPages = 3.
			m.EXPECT().GetTransactions(gomock.Any(), "test", 0, pageSize, timeEq(time.Time{}), timeEq(time.Time{})).Return(
				sampleTransactions[:20], 3, nil,
			)
			// Descent over the frozen window (to = ceiling), oldest page first.
			m.EXPECT().GetTransactions(gomock.Any(), "test", 2, pageSize, timeEq(time.Time{}), timeEq(ceiling)).Return(
				sampleTransactions[40:], 3, nil,
			)
			m.EXPECT().GetTransactions(gomock.Any(), "test", 1, pageSize, timeEq(time.Time{}), timeEq(ceiling)).Return(
				sampleTransactions[20:40], 3, nil,
			)
			m.EXPECT().GetTransactions(gomock.Any(), "test", 0, pageSize, timeEq(time.Time{}), timeEq(ceiling)).Return(
				sampleTransactions[:20], 3, nil,
			)

			// Mimic the engine loop: keep calling while HasMore, threading NewState.
			state := []byte(`{}`)
			allPayments := make([]models.PSPPayment, 0)
			calls := 0
			for {
				resp, err := plg.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{
					State:       state,
					PageSize:    pageSize,
					FromPayload: []byte(`{"reference": "test"}`),
				})
				Expect(err).To(BeNil())
				allPayments = append(allPayments, resp.Payments...)
				state = resp.NewState
				calls++

				if calls == 1 {
					// open: ceiling frozen, descent set to start at the oldest page, nothing
					// emitted yet, watermark not advanced.
					Expect(resp.HasMore).To(BeTrue())
					Expect(resp.Payments).To(HaveLen(0))
					var mid paymentsState
					Expect(json.Unmarshal(state, &mid)).To(BeNil())
					Expect(mid.NextPage).To(Equal(2))
					Expect(mid.Ceiling.UTC()).To(Equal(ceiling.UTC()))
					Expect(mid.LastTransactionTime.IsZero()).To(BeTrue())
				}

				if !resp.HasMore {
					break
				}
				Expect(calls).To(BeNumerically("<", 6)) // guard against an infinite loop
			}

			Expect(calls).To(Equal(4)) // open + 3 page emits
			// Nothing lost despite the backlog exceeding PageSize.
			Expect(allPayments).To(HaveLen(50))
			// Emitted strictly oldest-first across the whole multi-page window.
			expectOldestFirst(allPayments)

			var final paymentsState
			Expect(json.Unmarshal(state, &final)).To(BeNil())
			Expect(final.Ceiling.IsZero()).To(BeTrue())
			Expect(final.NextPage).To(Equal(0))
			Expect(final.LastTransactionTime.UTC()).To(Equal(ceiling.UTC()))
		})

		It("should fetch only transactions newer than the existing watermark", func(ctx SpecContext) {
			// Watermark sits at index 10's TransactionDate; the API (newest-first) returns
			// everything >= the watermark, i.e. indices 0..10, on a single page.
			watermark, _ := time.Parse(transactionDateLayout, sampleTransactions[10].TransactionDate)
			req := models.FetchNextPaymentsRequest{
				State: []byte(fmt.Sprintf(
					`{"lastTransactionTime":"%s","version":%d}`,
					watermark.UTC().Format(time.RFC3339Nano), paymentsStateVersion,
				)),
				PageSize:    40,
				FromPayload: []byte(`{"reference": "test"}`),
			}

			m.EXPECT().GetTransactions(gomock.Any(), "test", 0, 40, timeEq(watermark), timeEq(time.Time{})).Return(
				sampleTransactions[:11],
				1,
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			// Index 10 (== watermark) is skipped; indices 0..9 are kept.
			Expect(resp.Payments).To(HaveLen(10))
			Expect(resp.HasMore).To(BeFalse())
			expectOldestFirst(resp.Payments)

			var state paymentsState
			Expect(json.Unmarshal(resp.NewState, &state)).To(BeNil())
			Expect(state.Ceiling.IsZero()).To(BeTrue())
			newest, _ := time.Parse(transactionDateLayout, sampleTransactions[0].TransactionDate)
			Expect(state.LastTransactionTime.UTC()).To(Equal(newest.UTC()))
		})

		It("should restart at page 0 when state has a stale page but no ceiling", func(ctx SpecContext) {
			// Defensive: a corrupt/legacy state with a non-zero NextPage but a zero Ceiling
			// must be treated as a fresh window (peek page 0), never paged mid-window with
			// a zero ceiling.
			req := models.FetchNextPaymentsRequest{
				State:       []byte(fmt.Sprintf(`{"nextPage":7,"version":%d}`, paymentsStateVersion)),
				PageSize:    60,
				FromPayload: []byte(`{"reference": "test"}`),
			}

			m.EXPECT().GetTransactions(gomock.Any(), "test", 0, 60, timeEq(time.Time{}), timeEq(time.Time{})).Return(
				sampleTransactions,
				1,
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(50))
			Expect(resp.HasMore).To(BeFalse())

			var state paymentsState
			Expect(json.Unmarshal(resp.NewState, &state)).To(BeNil())
			Expect(state.NextPage).To(Equal(0))
			Expect(state.Ceiling.IsZero()).To(BeTrue())
			newest, _ := time.Parse(transactionDateLayout, sampleTransactions[0].TransactionDate)
			Expect(state.LastTransactionTime.UTC()).To(Equal(newest.UTC()))
		})

		It("should reset stale pre-version state (migration) and refetch from zero", func(ctx SpecContext) {
			// Pre-fix state: a PostedDate-derived watermark and no version field (version 0).
			stale, _ := time.Parse(transactionDateLayout, sampleTransactions[5].PostedDate)
			req := models.FetchNextPaymentsRequest{
				State: []byte(fmt.Sprintf(
					`{"lastTransactionTime":"%s"}`, stale.UTC().Format(time.RFC3339Nano),
				)),
				PageSize:    60,
				FromPayload: []byte(`{"reference": "test"}`),
			}

			// Migration must discard the stale watermark and refetch from zero.
			m.EXPECT().GetTransactions(gomock.Any(), "test", 0, 60, timeEq(time.Time{}), timeEq(time.Time{})).Return(
				sampleTransactions,
				1,
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(50))

			var state paymentsState
			Expect(json.Unmarshal(resp.NewState, &state)).To(BeNil())
			Expect(state.Version).To(Equal(paymentsStateVersion))
			newest, _ := time.Parse(transactionDateLayout, sampleTransactions[0].TransactionDate)
			Expect(state.LastTransactionTime.UTC()).To(Equal(newest.UTC()))
		})
	})
})

// expectOldestFirst asserts payments are emitted in non-decreasing CreatedAt order, so the
// engine seeds each payment's base row from the earliest event.
func expectOldestFirst(payments []models.PSPPayment) {
	for i := 1; i < len(payments); i++ {
		Expect(payments[i-1].CreatedAt.After(payments[i].CreatedAt)).To(BeFalse())
	}
}

// timeEq matches a time.Time argument by instant (time.Equal), ignoring location and
// monotonic-clock differences that survive JSON round-trips through connector state.
type timeEqMatcher struct{ t time.Time }

func (m timeEqMatcher) Matches(x any) bool {
	t, ok := x.(time.Time)
	return ok && t.Equal(m.t)
}

func (m timeEqMatcher) String() string {
	return fmt.Sprintf("is the time %s", m.t)
}

func timeEq(t time.Time) gomock.Matcher {
	return timeEqMatcher{t: t}
}

var _ = Describe("Modulr Plugin Transaction to Payments", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  *Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{}
		plg.client = m
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("transaction to payments", func() {
		var (
			samplePayinTransaction    client.Transaction
			samplePayoutTransaction   client.Transaction
			sampleTransferTransaction client.Transaction
			sampleTransfer            client.TransferResponse
			now                       time.Time
		)

		BeforeEach(func() {
			now, _ = time.Parse("2006-01-02T15:04:05.999-0700", time.Now().UTC().Format("2006-01-02T15:04:05.999-0700"))

			sampleTransfer = client.TransferResponse{
				ID:                "test",
				CreatedDate:       now.UTC().Format("2006-01-02T15:04:05.999-0700"),
				ExternalReference: "test1",
				ApprovalStatus:    "test1",
				Status:            "PROCESSED",
				Details: client.Details{
					SourceAccountID: "acc1",
					Destination: client.Destination{
						ID: "acc2",
					},
					Currency: "EUR",
					Amount:   "150.01",
				},
			}

			samplePayinTransaction = client.Transaction{
				ID:              "1",
				Type:            "PI_FAST",
				Amount:          "130.00",
				Credit:          true,
				SourceID:        "PI1",
				Description:     "test1",
				PostedDate:      now.UTC().Format("2006-01-02T15:04:05.999-0700"),
				TransactionDate: now.UTC().Format("2006-01-02T15:04:05.999-0700"),
				Account: client.Account{
					Currency: "USD",
				},
			}

			samplePayoutTransaction = client.Transaction{
				ID:              "2",
				Type:            "PO_FAST",
				Amount:          "145.08",
				Credit:          false,
				SourceID:        "PO2",
				Description:     "test2",
				PostedDate:      now.UTC().Format("2006-01-02T15:04:05.999-0700"),
				TransactionDate: now.UTC().Format("2006-01-02T15:04:05.999-0700"),
				Account: client.Account{
					Currency: "USD",
				},
			}

			sampleTransferTransaction = client.Transaction{
				ID:              "3",
				Type:            "INT_INTERC",
				Amount:          "150.01",
				Credit:          true,
				SourceID:        "test",
				Description:     "test3",
				PostedDate:      now.UTC().Format("2006-01-02T15:04:05.999-0700"),
				TransactionDate: now.UTC().Format("2006-01-02T15:04:05.999-0700"),
				Account: client.Account{
					Currency: "EUR",
				},
			}
		})

		It("should return an error - wrong amount string", func(ctx SpecContext) {
			po := samplePayoutTransaction
			po.Amount = "wrong"
			payment, err := plg.transactionToPayment(ctx, po, models.PSPAccount{})
			Expect(err).ToNot(BeNil())
			Expect(payment).To(BeNil())
			_ = samplePayinTransaction
			_ = sampleTransferTransaction
			_ = sampleTransfer
		})

		It("should return an error - wrong posted date", func(ctx SpecContext) {
			po := samplePayoutTransaction
			po.PostedDate = "wrong"
			payment, err := plg.transactionToPayment(ctx, po, models.PSPAccount{})
			Expect(err).ToNot(BeNil())
			Expect(payment).To(BeNil())
		})

		It("should return an error - fetch transfer error", func(ctx SpecContext) {
			m.EXPECT().GetTransfer(ctx, "test").Return(
				client.TransferResponse{},
				errors.New("test error"),
			)

			payment, err := plg.fetchAndTranslateTransfer(ctx, sampleTransferTransaction)
			Expect(err).ToNot(BeNil())
			Expect(payment).To(BeNil())
		})

		It("should return a nil payment - unhandled currency", func(ctx SpecContext) {
			po := samplePayoutTransaction
			po.Account.Currency = "HUF"
			payment, err := plg.transactionToPayment(ctx, po, models.PSPAccount{})
			Expect(err).To(BeNil())
			Expect(payment).To(BeNil())
		})

		It("should return a payin payment", func(ctx SpecContext) {
			payment, err := plg.transactionToPayment(ctx, samplePayinTransaction, models.PSPAccount{Reference: "acc1"})
			Expect(err).To(BeNil())
			Expect(payment).ToNot(BeNil())

			expected := models.PSPPayment{
				Reference:                   samplePayinTransaction.SourceID,
				CreatedAt:                   now,
				Type:                        models.PAYMENT_TYPE_PAYIN,
				Amount:                      big.NewInt(13000),
				Asset:                       "USD/2",
				Scheme:                      models.PAYMENT_SCHEME_OTHER,
				Status:                      models.PAYMENT_STATUS_SUCCEEDED,
				DestinationAccountReference: pointer.For("acc1"),
			}

			comparePSPPayments(*payment, expected)
		})

		It("should return a payout payment", func(ctx SpecContext) {
			payment, err := plg.transactionToPayment(ctx, samplePayoutTransaction, models.PSPAccount{Reference: "acc2"})
			Expect(err).To(BeNil())
			Expect(payment).ToNot(BeNil())

			expected := models.PSPPayment{
				Reference:              samplePayoutTransaction.SourceID,
				CreatedAt:              now,
				Type:                   models.PAYMENT_TYPE_PAYOUT,
				Amount:                 big.NewInt(14508),
				Asset:                  "USD/2",
				Scheme:                 models.PAYMENT_SCHEME_OTHER,
				Status:                 models.PAYMENT_STATUS_SUCCEEDED,
				SourceAccountReference: pointer.For("acc2"),
			}

			comparePSPPayments(*payment, expected)
		})

		It("should return a transfer payment", func(ctx SpecContext) {
			m.EXPECT().GetTransfer(gomock.Any(), "test").Return(
				sampleTransfer,
				nil,
			)

			payment, err := plg.transactionToPayment(ctx, sampleTransferTransaction, models.PSPAccount{Reference: "acc1"})
			Expect(err).To(BeNil())
			Expect(payment).ToNot(BeNil())

			expected := models.PSPPayment{
				Reference:                   sampleTransferTransaction.SourceID,
				CreatedAt:                   now,
				Type:                        models.PAYMENT_TYPE_TRANSFER,
				Amount:                      big.NewInt(15001),
				Asset:                       "EUR/2",
				Scheme:                      models.PAYMENT_SCHEME_OTHER,
				Status:                      models.PAYMENT_STATUS_SUCCEEDED,
				SourceAccountReference:      pointer.For("acc1"),
				DestinationAccountReference: pointer.For("acc2"),
			}

			comparePSPPayments(*payment, expected)
		})

		It("should return a nil payment - transfer in debit", func(ctx SpecContext) {
			tr := sampleTransferTransaction
			tr.Credit = false

			payment, err := plg.transactionToPayment(ctx, tr, models.PSPAccount{Reference: "acc1"})
			Expect(err).To(BeNil())
			Expect(payment).To(BeNil())
		})
	})
})

func comparePSPPayments(a, b models.PSPPayment) {
	Expect(a.ParentReference).To(Equal(b.ParentReference))
	Expect(a.Reference).To(Equal(b.Reference))
	Expect(a.CreatedAt.UTC()).To(Equal(b.CreatedAt.UTC()))
	Expect(a.Type).To(Equal(b.Type))
	Expect(a.Amount).To(Equal(b.Amount))
	Expect(a.Asset).To(Equal(b.Asset))
	Expect(a.Scheme).To(Equal(b.Scheme))
	Expect(a.Status).To(Equal(b.Status))

	switch {
	case a.SourceAccountReference != nil && b.SourceAccountReference != nil:
		Expect(*a.SourceAccountReference).To(Equal(*b.SourceAccountReference))
	case a.SourceAccountReference == nil && b.SourceAccountReference == nil:
	default:
		Fail(fmt.Sprintf("SourceAccountReference mismatch: %v != %v", a.SourceAccountReference, b.SourceAccountReference))
	}

	switch {
	case a.DestinationAccountReference != nil && b.DestinationAccountReference != nil:
		Expect(*a.DestinationAccountReference).To(Equal(*b.DestinationAccountReference))
	case a.DestinationAccountReference == nil && b.DestinationAccountReference == nil:
	default:
		Fail(fmt.Sprintf("DestinationAccountReference mismatch: %v != %v", a.DestinationAccountReference, b.DestinationAccountReference))
	}

	Expect(len(a.Metadata)).To(Equal(len(b.Metadata)))
	for k, v := range a.Metadata {
		Expect(v).To(Equal(b.Metadata[k]))
	}
}
