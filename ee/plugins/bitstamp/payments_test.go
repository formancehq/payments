package bitstamp

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Bitstamp Plugin Payments", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  *Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{
			Plugin: plugins.NewBasePlugin(),
			client: m,
			logger: logging.NewDefaultLogger(GinkgoWriter, true, false, false),
			currencies: map[string]int{
				"USD": 2,
				"EUR": 2,
				"BTC": 8,
				"ETH": 18,
			},
			currLastSync: time.Now(),
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetching next payments", func() {
		It("should return an error - get transactions error", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 100,
			}

			m.EXPECT().GetUserTransactions(gomock.Any(), gomock.Nil(), 100).Return(
				nil,
				errors.New("test error"),
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should fetch deposit transactions as PAYIN", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 100,
			}

			m.EXPECT().GetUserTransactions(gomock.Any(), gomock.Nil(), 100).Return(
				[]client.UserTransaction{
					{
						ID:       12345,
						Datetime: "2024-01-15 10:30:00.000000",
						Type:     "0",
						Fee:      "0.00",
						CurrencyAmounts: map[string]string{
							"btc": "0.50000000",
						},
					},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			Expect(resp.HasMore).To(BeFalse())

			payment := resp.Payments[0]
			Expect(payment.Reference).To(Equal("12345"))
			Expect(payment.Type).To(Equal(models.PAYMENT_TYPE_PAYIN))
			Expect(payment.Status).To(Equal(models.PAYMENT_STATUS_SUCCEEDED))
			Expect(payment.Asset).To(Equal("BTC/8"))
			// 0.50000000 BTC = 50000000 satoshis
			Expect(payment.Amount.Int64()).To(Equal(int64(50000000)))
			Expect(payment.Metadata[MetadataPrefix+"type"]).To(Equal("0"))
		})

		It("should fetch withdrawal transactions as PAYOUT", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 100,
			}

			m.EXPECT().GetUserTransactions(gomock.Any(), gomock.Nil(), 100).Return(
				[]client.UserTransaction{
					{
						ID:       12346,
						Datetime: "2024-01-15 11:00:00.000000",
						Type:     "1",
						Fee:      "0.0005",
						CurrencyAmounts: map[string]string{
							"btc": "-0.25000000",
						},
					},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))

			payment := resp.Payments[0]
			Expect(payment.Reference).To(Equal("12346"))
			Expect(payment.Type).To(Equal(models.PAYMENT_TYPE_PAYOUT))
			Expect(payment.Asset).To(Equal("BTC/8"))
			// Amount should be positive (absolute value)
			Expect(payment.Amount.Int64()).To(Equal(int64(25000000)))
			Expect(payment.Metadata[MetadataPrefix+"fee"]).To(Equal("0.0005"))
		})

		It("should skip order transactions", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 100,
			}

			m.EXPECT().GetUserTransactions(gomock.Any(), gomock.Nil(), 100).Return(
				[]client.UserTransaction{
					{
						ID:       12347,
						Datetime: "2024-01-15 12:00:00.000000",
						Type:     "36",
						Fee:      "1.25",
						OrderID:  99999,
						CurrencyAmounts: map[string]string{
							"btc": "-0.10000000",
							"usd": "4500.00",
						},
					},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(0))
		})

		It("should skip legacy market trade transactions", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 100,
			}

			m.EXPECT().GetUserTransactions(gomock.Any(), gomock.Nil(), 100).Return(
				[]client.UserTransaction{
					{
						ID:       12352,
						Datetime: "2024-01-15 12:00:00.000000",
						Type:     "2",
						CurrencyAmounts: map[string]string{
							"btc": "-0.10000000",
							"usd": "4500.00",
						},
					},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(0))
		})

		It("should fetch sub-account transfer as TRANSFER", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 100,
			}

			m.EXPECT().GetUserTransactions(gomock.Any(), gomock.Nil(), 100).Return(
				[]client.UserTransaction{
					{
						ID:       12348,
						Datetime: "2024-01-15 13:00:00.000000",
						Type:     "14",
						Fee:      "0",
						CurrencyAmounts: map[string]string{
							"usd": "1000.00",
						},
					},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))

			payment := resp.Payments[0]
			Expect(payment.Type).To(Equal(models.PAYMENT_TYPE_TRANSFER))
		})

		It("should fetch staking rewards as PAYIN", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 100,
			}

			m.EXPECT().GetUserTransactions(gomock.Any(), gomock.Nil(), 100).Return(
				[]client.UserTransaction{
					{
						ID:       12353,
						Datetime: "2024-01-15 13:30:00.000000",
						Type:     "27",
						CurrencyAmounts: map[string]string{
							"eth": "0.010000000000000000",
						},
					},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			Expect(resp.Payments[0].Type).To(Equal(models.PAYMENT_TYPE_PAYIN))
		})

		It("should skip transactions with no matching currency", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 100,
			}

			m.EXPECT().GetUserTransactions(gomock.Any(), gomock.Nil(), 100).Return(
				[]client.UserTransaction{
					{
						ID:       12349,
						Datetime: "2024-01-15 14:00:00.000000",
						Type:     "0",
						Fee:      "0",
						CurrencyAmounts: map[string]string{
							"unknown_coin": "100.00",
						},
					},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(0))
		})

		It("should skip payment transactions with multiple non-zero assets", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 100,
			}

			m.EXPECT().GetUserTransactions(gomock.Any(), gomock.Nil(), 100).Return(
				[]client.UserTransaction{
					{
						ID:       12354,
						Datetime: "2024-01-15 14:30:00.000000",
						Type:     "0",
						CurrencyAmounts: map[string]string{
							"btc": "0.10000000",
							"usd": "4500.00",
						},
					},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(0))
		})

		It("should use since_id pagination with req.PageSize", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{"lastTransactionID": 100}`),
				PageSize: 50,
			}

			m.EXPECT().GetUserTransactions(gomock.Any(), ptrInt64(100), 50).Return(
				[]client.UserTransaction{
					{
						ID:       12350,
						Datetime: "2024-01-15 15:00:00.000000",
						Type:     "0",
						Fee:      "0",
						CurrencyAmounts: map[string]string{
							"btc": "1.00000000",
						},
					},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			Expect(resp.HasMore).To(BeFalse()) // 1 < 50

			// Verify state advances to the highest returned transaction ID.
			var newState paymentsState
			err = json.Unmarshal(resp.NewState, &newState)
			Expect(err).To(BeNil())
			Expect(newState.LastTransactionID).To(Equal(int64(12350)))
		})

		It("should report HasMore when page is full", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 2,
			}

			m.EXPECT().GetUserTransactions(gomock.Any(), gomock.Nil(), 2).Return(
				[]client.UserTransaction{
					{
						ID:       1,
						Datetime: "2024-01-15 10:00:00.000000",
						Type:     "0",
						Fee:      "0",
						CurrencyAmounts: map[string]string{
							"btc": "1.00000000",
						},
					},
					{
						ID:       2,
						Datetime: "2024-01-15 11:00:00.000000",
						Type:     "0",
						Fee:      "0",
						CurrencyAmounts: map[string]string{
							"btc": "2.00000000",
						},
					},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(2))
			Expect(resp.HasMore).To(BeTrue())
		})

		It("should handle empty response", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 100,
			}

			m.EXPECT().GetUserTransactions(gomock.Any(), gomock.Nil(), 100).Return(
				[]client.UserTransaction{},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
		})
	})
})

func ptrInt64(v int64) *int64 {
	return &v
}
