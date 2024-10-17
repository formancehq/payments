package moneycorp

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/moneycorp/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Moneycorp Plugin Payments - check types and minor conversion", func() {
	var (
		plg *Plugin
	)

	Context("fetch next Payments", func() {
		var (
			m *client.MockClient

			samplePayments []*client.Transaction
			sampleTransfer *client.TransferResponse
			accRef         int32
			pageSize       int
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{client: m}

			pageSize = 5
			accRef = 3796
			sampleTransfer = &client.TransferResponse{
				ID: "transfer-1",
				Attributes: client.TransferAttributes{
					SendingAccountID:   1234,
					ReceivingAccountID: 4321,
					CreatedAt:          strings.TrimSuffix(time.Now().UTC().Format(time.RFC3339Nano), "Z"),
					UpdatedAt:          strings.TrimSuffix(time.Now().UTC().Format(time.RFC3339Nano), "Z"),
					TransferReference:  "test1",
					ClientReference:    "test1",
					TransferAmount:     json.Number("65"),
					TransferCurrency:   "EUR",
					TransferStatus:     "Cleared",
				},
			}
			samplePayments = []*client.Transaction{
				{
					ID: "transfer-1",
					Attributes: client.TransactionAttributes{
						AccountID: accRef,
						Type:      "Transfer",
						Direction: "Debit",
						Currency:  "EUR",
						Amount:    json.Number("65"),
						CreatedAt: strings.TrimSuffix(time.Now().UTC().Format(time.RFC3339Nano), "Z"),
					},
					Relationships: client.RelationShips{
						Data: client.Data{
							ID: sampleTransfer.ID,
						},
					},
				},
				{
					ID: "payment-1",
					Attributes: client.TransactionAttributes{
						AccountID: accRef,
						Type:      "Payment",
						Direction: "Debit",
						Currency:  "DKK",
						Amount:    json.Number("42"),
						CreatedAt: strings.TrimSuffix(time.Now().UTC().Format(time.RFC3339Nano), "Z"),
					},
					Relationships: client.RelationShips{
						Data: client.Data{
							ID: "Payout-1",
						},
					},
				},
				{
					ID: "exchange-1",
					Attributes: client.TransactionAttributes{
						AccountID: accRef,
						Type:      "Exchange",
						Direction: "Debit",
						Currency:  "GBP",
						Amount:    json.Number("28"),
						CreatedAt: strings.TrimSuffix(time.Now().UTC().Format(time.RFC3339Nano), "Z"),
					},
				},
				{
					ID: "charge-1",
					Attributes: client.TransactionAttributes{
						AccountID: accRef,
						Type:      "Charge",
						Direction: "Credit",
						Currency:  "JPY",
						Amount:    json.Number("6400"),
						CreatedAt: strings.TrimSuffix(time.Now().UTC().Format(time.RFC3339Nano), "Z"),
					},
				},
				{
					ID: "refund-1",
					Attributes: client.TransactionAttributes{
						AccountID: accRef,
						Type:      "Refund",
						Direction: "Credit",
						Currency:  "MAD",
						Amount:    json.Number("64"),
						CreatedAt: strings.TrimSuffix(time.Now().UTC().Format(time.RFC3339Nano), "Z"),
					},
				},
				{
					ID: "unsupported-1",
					Attributes: client.TransactionAttributes{
						AccountID: accRef,
						Type:      "Unsupported",
						Currency:  "USD",
						Amount:    json.Number("29"),
						CreatedAt: strings.TrimSuffix(time.Now().UTC().Format(time.RFC3339Nano), "Z"),
					},
				},
			}

		})

		It("fails when payments contain unsupported currencies", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				FromPayload: json.RawMessage(fmt.Sprintf(`{"reference": "%d"}`, accRef)),
				State:       json.RawMessage(`{}`),
				PageSize:    pageSize,
			}
			p := []*client.Transaction{
				{
					ID: "someid",
					Attributes: client.TransactionAttributes{
						Direction: "Debit",
						AccountID: accRef,
						Type:      "Payment",
						CreatedAt: strings.TrimSuffix(time.Now().UTC().Format(time.RFC3339Nano), "Z"),
						Currency:  "EEK",
					},
				},
			}
			m.EXPECT().GetTransactions(ctx, "3796", gomock.Any(), pageSize, gomock.Any()).Return(
				p,
				nil,
			)

			res, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(MatchError(currency.ErrMissingCurrencies))
			Expect(res.HasMore).To(BeFalse())
		})

		It("fetches payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				FromPayload: json.RawMessage(fmt.Sprintf(`{"reference": "%d"}`, accRef)),
				State:       json.RawMessage(`{}`),
				PageSize:    pageSize,
			}

			m.EXPECT().GetTransactions(ctx, "3796", gomock.Any(), pageSize, gomock.Any()).Return(
				samplePayments,
				nil,
			)

			m.EXPECT().GetTransfer(ctx, "3796", sampleTransfer.ID).Return(
				sampleTransfer,
				nil,
			)

			res, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Payments).To(HaveLen(len(samplePayments) - 1))
			Expect(res.HasMore).To(BeTrue())

			// Transfer
			Expect(res.Payments[0].Reference).To(Equal(samplePayments[0].ID))
			Expect(res.Payments[0].Type).To(Equal(models.PAYMENT_TYPE_TRANSFER))
			expectedAmount, err := samplePayments[0].Attributes.Amount.Int64()
			Expect(err).To(BeNil())
			Expect(res.Payments[0].Amount).To(Equal(big.NewInt(expectedAmount * 100))) // after conversion to minors
			// Payment
			Expect(res.Payments[1].Reference).To(Equal(samplePayments[1].Relationships.Data.ID))
			Expect(res.Payments[1].Type).To(Equal(models.PAYMENT_TYPE_PAYOUT))
			expectedAmount, err = samplePayments[1].Attributes.Amount.Int64()
			Expect(err).To(BeNil())
			Expect(res.Payments[1].Amount).To(Equal(big.NewInt(expectedAmount * 100))) // after conversion to minors
			// Exchange
			Expect(res.Payments[2].Reference).To(Equal(samplePayments[2].ID))
			Expect(res.Payments[2].Type).To(Equal(models.PAYMENT_TYPE_PAYOUT))
			expectedAmount, err = samplePayments[2].Attributes.Amount.Int64()
			Expect(err).To(BeNil())
			Expect(res.Payments[2].Amount).To(Equal(big.NewInt(expectedAmount * 100))) // after conversion to minors
			// Charge
			Expect(res.Payments[3].Reference).To(Equal(samplePayments[3].ID))
			Expect(res.Payments[3].Type).To(Equal(models.PAYMENT_TYPE_PAYIN))
			expectedAmount, err = samplePayments[3].Attributes.Amount.Int64()
			Expect(err).To(BeNil())
			Expect(res.Payments[3].Amount).To(Equal(big.NewInt(expectedAmount))) // currency already in minors
			// Refund
			Expect(res.Payments[4].Reference).To(Equal(samplePayments[4].ID))
			Expect(res.Payments[4].Type).To(Equal(models.PAYMENT_TYPE_PAYIN))
			expectedAmount, err = samplePayments[4].Attributes.Amount.Int64()
			Expect(err).To(BeNil())
			Expect(res.Payments[4].Amount).To(Equal(big.NewInt(expectedAmount * 100))) // after conversion to minors

		})
	})
})

var _ = Describe("Moneycorp Plugin Payments - check pagination", func() {
	var (
		plg *Plugin
	)

	Context("fetch next Payments", func() {
		var (
			m *client.MockClient

			samplePayments []*client.Transaction
			accRef         int32
			now            time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{client: m}
			accRef = 3796
			now = time.Now().UTC()

			samplePayments = make([]*client.Transaction, 0)
			for i := 0; i < 50; i++ {
				samplePayments = append(samplePayments, &client.Transaction{
					ID: fmt.Sprintf("transaction-%d", i),
					Attributes: client.TransactionAttributes{
						AccountID: accRef,
						Type:      "Payment",
						Direction: "Debit",
						Currency:  "EUR",
						Amount:    json.Number("42"),
						CreatedAt: strings.TrimSuffix(now.Add(-time.Duration(60-i)*time.Minute).UTC().Format(time.RFC3339Nano), "Z"),
					},
					Relationships: client.RelationShips{
						Data: client.Data{
							ID: fmt.Sprintf("%d", i),
						},
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
				FromPayload: json.RawMessage(fmt.Sprintf(`{"reference": "%d"}`, accRef)),
			}

			m.EXPECT().GetTransactions(ctx, fmt.Sprintf("%d", accRef), 0, 60, time.Time{}).Return(
				[]*client.Transaction{},
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
				FromPayload: json.RawMessage(fmt.Sprintf(`{"reference": "%d"}`, accRef)),
			}

			m.EXPECT().GetTransactions(ctx, fmt.Sprintf("%d", accRef), 0, 60, time.Time{}).Return(
				[]*client.Transaction{},
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
			Expect(state.LastCreatedAt.IsZero()).To(BeTrue())
		})

		It("should fetch next payments - no state pageSize > total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:       []byte(`{}`),
				PageSize:    60,
				FromPayload: json.RawMessage(fmt.Sprintf(`{"reference": "%d"}`, accRef)),
			}

			m.EXPECT().GetTransactions(ctx, fmt.Sprintf("%d", accRef), 0, 60, time.Time{}).Return(
				samplePayments,
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
			createdAtTime, _ := time.Parse(time.RFC3339Nano, samplePayments[49].Attributes.CreatedAt+"Z")
			Expect(state.LastCreatedAt.UTC()).To(Equal(createdAtTime.UTC()))
		})

		It("should fetch next payments - no state pageSize < total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:       []byte(`{}`),
				PageSize:    40,
				FromPayload: json.RawMessage(fmt.Sprintf(`{"reference": "%d"}`, accRef)),
			}

			m.EXPECT().GetTransactions(ctx, fmt.Sprintf("%d", accRef), 0, 40, time.Time{}).Return(
				samplePayments[:40],
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
			createdAtTime, _ := time.Parse(time.RFC3339Nano, samplePayments[39].Attributes.CreatedAt+"Z")
			Expect(state.LastCreatedAt.UTC()).To(Equal(createdAtTime.UTC()))
		})

		It("should fetch next payments - with state pageSize < total payments", func(ctx SpecContext) {
			lastCreatedAt, _ := time.Parse(time.RFC3339Nano, samplePayments[38].Attributes.CreatedAt+"Z")
			req := models.FetchNextPaymentsRequest{
				State:       []byte(fmt.Sprintf(`{"lastCreatedAt": "%s"}`, lastCreatedAt.Format(time.RFC3339Nano))),
				PageSize:    40,
				FromPayload: json.RawMessage(fmt.Sprintf(`{"reference": "%d"}`, accRef)),
			}

			m.EXPECT().GetTransactions(ctx, fmt.Sprintf("%d", accRef), 0, 40, lastCreatedAt.UTC()).Return(
				samplePayments[:40],
				nil,
			)

			m.EXPECT().GetTransactions(ctx, fmt.Sprintf("%d", accRef), 1, 40, lastCreatedAt.UTC()).Return(
				samplePayments[41:],
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
			createdAtTime, _ := time.Parse(time.RFC3339Nano, samplePayments[49].Attributes.CreatedAt+"Z")
			Expect(state.LastCreatedAt.UTC()).To(Equal(createdAtTime.UTC()))
		})
	})
})
