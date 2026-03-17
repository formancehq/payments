package krakenpro

import (
	"encoding/json"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Krakenpro Payments", func() {
	var (
		p      *Plugin
		m      *client.MockClient
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		p = &Plugin{
			Plugin: plugins.NewBasePlugin(),
			client: m,
			logger: logger,
			config: Config{
				APIKey: "test-api-key",
			},
			accountRef: "kraken-test12345",
			currencies: map[string]int{"USD": 2, "BTC": 8, "ETH": 18},
		}
	})

	Context("fetch next payments", func() {
		It("should map ledger entries to payments", func(ctx SpecContext) {
			m.EXPECT().GetLedgers(gomock.Any(), 0, gomock.Any()).Return(
				&client.LedgersResponse{
					Error: nil,
					Result: client.LedgersResult{
						Ledgers: map[string]client.LedgerEntry{
							"L1234-ABCDE": {
								RefID:   "TJKLMN-12345",
								Time:    1617331200.0,
								Type:    "deposit",
								Asset:   "XXBT",
								Amount:  "0.12345678",
								Fee:     "0.00000000",
								Balance: "1.00000000",
							},
						},
						Count: 1,
					},
				},
				nil,
			)

			req := models.FetchNextPaymentsRequest{
				State:    json.RawMessage(`{}`),
				PageSize: 50,
			}

			resp, err := p.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))

			payment := resp.Payments[0]
			Expect(payment.Reference).To(Equal("L1234-ABCDE"))
			Expect(payment.Type).To(Equal(models.PAYMENT_TYPE_PAYIN))
			Expect(payment.Status).To(Equal(models.PAYMENT_STATUS_SUCCEEDED))
			Expect(payment.Metadata["type"]).To(Equal("deposit"))
			Expect(payment.Metadata["refid"]).To(Equal("TJKLMN-12345"))
			Expect(payment.Raw).ToNot(BeNil())
			Expect(resp.HasMore).To(BeFalse())
		})

		It("should handle negative amounts (withdrawals)", func(ctx SpecContext) {
			m.EXPECT().GetLedgers(gomock.Any(), 0, gomock.Any()).Return(
				&client.LedgersResponse{
					Error: nil,
					Result: client.LedgersResult{
						Ledgers: map[string]client.LedgerEntry{
							"L5678-FGHIJ": {
								RefID:   "TWITHDRAW-1",
								Time:    1617331200.0,
								Type:    "withdrawal",
								Asset:   "ZUSD",
								Amount:  "-500.00",
								Fee:     "5.00",
								Balance: "170788.6158",
							},
						},
						Count: 1,
					},
				},
				nil,
			)

			req := models.FetchNextPaymentsRequest{
				State:    json.RawMessage(`{}`),
				PageSize: 50,
			}

			resp, err := p.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))

			payment := resp.Payments[0]
			Expect(payment.Type).To(Equal(models.PAYMENT_TYPE_PAYOUT))
			Expect(payment.Metadata["fee"]).To(Equal("5.00"))
		})

		It("should handle offset pagination", func(ctx SpecContext) {
			// Create 50 entries to trigger HasMore
			entries := make(map[string]client.LedgerEntry)
			for i := range 50 {
				key := "L" + string(rune('A'+i%26)) + string(rune('0'+i/26))
				entries[key] = client.LedgerEntry{
					Time:   1617331200.0,
					Type:   "trade",
					Asset:  "XXBT",
					Amount: "0.001",
					Fee:    "0.0000",
				}
			}

			m.EXPECT().GetLedgers(gomock.Any(), 0, gomock.Any()).Return(
				&client.LedgersResponse{
					Error: nil,
					Result: client.LedgersResult{
						Ledgers: entries,
						Count:   100,
					},
				},
				nil,
			)

			req := models.FetchNextPaymentsRequest{
				State:    json.RawMessage(`{}`),
				PageSize: 50,
			}

			resp, err := p.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.HasMore).To(BeTrue())

			// Verify state has offset
			var newState paymentsState
			Expect(json.Unmarshal(resp.NewState, &newState)).To(Succeed())
			Expect(newState.Offset).To(Equal(50))
			Expect(newState.LastSeenTime).ToNot(BeZero())
		})
	})

	Context("ledger type mapping", func() {
		DescribeTable("should map all 16 types correctly",
			func(ledgerType string, expectedType models.PaymentType) {
				Expect(ledgerTypeToPaymentType(ledgerType)).To(Equal(expectedType))
			},
			Entry("deposit → PAYIN", "deposit", models.PAYMENT_TYPE_PAYIN),
			Entry("withdrawal → PAYOUT", "withdrawal", models.PAYMENT_TYPE_PAYOUT),
			Entry("trade → TRANSFER", "trade", models.PAYMENT_TYPE_TRANSFER),
			Entry("transfer → TRANSFER", "transfer", models.PAYMENT_TYPE_TRANSFER),
			Entry("margin → OTHER", "margin", models.PaymentType(models.PAYMENT_TYPE_OTHER)),
			Entry("rollover → OTHER", "rollover", models.PaymentType(models.PAYMENT_TYPE_OTHER)),
			Entry("spend → PAYOUT", "spend", models.PAYMENT_TYPE_PAYOUT),
			Entry("receive → PAYIN", "receive", models.PAYMENT_TYPE_PAYIN),
			Entry("settled → OTHER", "settled", models.PaymentType(models.PAYMENT_TYPE_OTHER)),
			Entry("adjustment → OTHER", "adjustment", models.PaymentType(models.PAYMENT_TYPE_OTHER)),
			Entry("staking → PAYIN", "staking", models.PAYMENT_TYPE_PAYIN),
			Entry("sale → PAYOUT", "sale", models.PAYMENT_TYPE_PAYOUT),
			Entry("dividend → PAYIN", "dividend", models.PAYMENT_TYPE_PAYIN),
			Entry("nft_trade → TRANSFER", "nft_trade", models.PAYMENT_TYPE_TRANSFER),
			Entry("nft_rebate → PAYIN", "nft_rebate", models.PAYMENT_TYPE_PAYIN),
			Entry("credit → PAYIN", "credit", models.PAYMENT_TYPE_PAYIN),
			Entry("unknown → OTHER", "some_future_type", models.PaymentType(models.PAYMENT_TYPE_OTHER)),
		)
	})
})
