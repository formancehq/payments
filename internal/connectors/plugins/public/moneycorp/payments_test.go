package moneycorp

import (
	"encoding/json"
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

var _ = Describe("Moneycorp Plugin Payments", func() {
	var (
		plg *Plugin
	)

	Context("fetch next Payments", func() {
		var (
			m *client.MockClient

			samplePayments []*client.Transaction
			accRef         string
			pageSize       int
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{client: m}

			pageSize = 5
			accRef = "baseAcc"
			samplePayments = []*client.Transaction{
				{
					ID: "transfer-1",
					Attributes: client.TransactionAttributes{
						Type:      "Transfer",
						Currency:  "EUR",
						Amount:    json.Number("65"),
						CreatedAt: strings.TrimSuffix(time.Now().UTC().Format(time.RFC3339Nano), "Z"),
					},
				},
				{
					ID: "payment-1",
					Attributes: client.TransactionAttributes{
						Type:      "Payment",
						Direction: "Debit",
						Currency:  "DKK",
						Amount:    json.Number("42"),
						CreatedAt: strings.TrimSuffix(time.Now().UTC().Format(time.RFC3339Nano), "Z"),
					},
				},
				{
					ID: "exchange-1",
					Attributes: client.TransactionAttributes{
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
				FromPayload: json.RawMessage(fmt.Sprintf(`{"reference": "%s"}`, accRef)),
				State:       json.RawMessage(`{}`),
				PageSize:    pageSize,
			}
			p := []*client.Transaction{
				{
					ID: "someid",
					Attributes: client.TransactionAttributes{
						Type:      "Transfer",
						CreatedAt: strings.TrimSuffix(time.Now().UTC().Format(time.RFC3339Nano), "Z"),
						Currency:  "EEK",
					},
				},
			}
			m.EXPECT().GetTransactions(ctx, accRef, gomock.Any(), pageSize, gomock.Any()).Return(
				p,
				nil,
			)
			res, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(MatchError(currency.ErrMissingCurrencies))
			Expect(res.HasMore).To(BeFalse())
		})

		It("fetches payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				FromPayload: json.RawMessage(fmt.Sprintf(`{"reference": "%s"}`, accRef)),
				State:       json.RawMessage(`{}`),
				PageSize:    pageSize,
			}
			m.EXPECT().GetTransactions(ctx, accRef, gomock.Any(), pageSize, gomock.Any()).Return(
				samplePayments,
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
			Expect(res.Payments[1].Reference).To(Equal(samplePayments[1].ID))
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
