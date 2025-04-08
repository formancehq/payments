package modulr

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/modulr/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Modulr Plugin Payments", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("fetching next accounts", func() {
		var (
			m                  *client.MockClient
			sampleTransactions []client.Transaction
			now                time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
			now = time.Now().UTC()

			sampleTransactions = make([]client.Transaction, 0)
			for i := 0; i < 50; i++ {
				sampleTransactions = append(sampleTransactions, client.Transaction{
					ID:              fmt.Sprintf("%d", i),
					Type:            "PI_FAST",
					Amount:          "100.01",
					Credit:          false,
					SourceID:        fmt.Sprintf("test-%d", i),
					Description:     fmt.Sprintf("Description %d", i),
					PostedDate:      now.Add(-time.Duration(50-i) * time.Minute).UTC().Format("2006-01-02T15:04:05.999-0700"),
					TransactionDate: now.Add(-time.Duration(50-i) * time.Minute).UTC().Format("2006-01-02T15:04:05.999-0700"),
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

			m.EXPECT().GetTransactions(gomock.Any(), "test", 0, 60, time.Time{}).Return(
				[]client.Transaction{},
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

			m.EXPECT().GetTransactions(gomock.Any(), "test", 0, 60, time.Time{}).Return(
				[]client.Transaction{},
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
			Expect(state.LastTransactionTime.IsZero()).To(BeTrue())
		})

		It("should fetch next payments - no state pageSize > total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:       []byte(`{}`),
				PageSize:    60,
				FromPayload: []byte(`{"reference": "test"}`),
			}

			m.EXPECT().GetTransactions(gomock.Any(), "test", 0, 60, time.Time{}).Return(
				sampleTransactions,
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
			createdTime, _ := time.Parse("2006-01-02T15:04:05.999-0700", sampleTransactions[49].PostedDate)
			Expect(state.LastTransactionTime.UTC()).To(Equal(createdTime.UTC()))
		})

		It("should fetch next payments - no state pageSize < total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:       []byte(`{}`),
				PageSize:    40,
				FromPayload: []byte(`{"reference": "test"}`),
			}

			m.EXPECT().GetTransactions(gomock.Any(), "test", 0, 40, time.Time{}).Return(
				sampleTransactions[:40],
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
			createdTime, _ := time.Parse("2006-01-02T15:04:05.999-0700", sampleTransactions[39].PostedDate)
			Expect(state.LastTransactionTime.UTC()).To(Equal(createdTime.UTC()))
		})

		It("should fetch next payments - with state pageSize < total payments", func(ctx SpecContext) {
			lastCreatedAt, _ := time.Parse("2006-01-02T15:04:05.999-0700", sampleTransactions[38].PostedDate)
			req := models.FetchNextPaymentsRequest{
				State:       []byte(fmt.Sprintf(`{"lastTransactionTime": "%s"}`, lastCreatedAt.UTC().Format(time.RFC3339Nano))),
				PageSize:    40,
				FromPayload: []byte(`{"reference": "test"}`),
			}

			m.EXPECT().GetTransactions(gomock.Any(), "test", 0, 40, lastCreatedAt.UTC()).Return(
				sampleTransactions[:40],
				nil,
			)

			m.EXPECT().GetTransactions(gomock.Any(), "test", 1, 40, lastCreatedAt.UTC()).Return(
				sampleTransactions[41:],
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
			createdTime, _ := time.Parse("2006-01-02T15:04:05.999-0700", sampleTransactions[49].PostedDate)
			Expect(state.LastTransactionTime.UTC()).To(Equal(createdTime.UTC()))
		})
	})
})

var _ = Describe("Modulr Plugin Transaction to Payments", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("fetching next accounts", func() {
		var (
			m                         *client.MockClient
			samplePayinTransaction    client.Transaction
			samplePayoutTransaction   client.Transaction
			sampleTransferTransaction client.Transaction
			sampleTransfer            client.TransferResponse
			now                       time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
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
