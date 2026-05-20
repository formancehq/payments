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
		// Three-source orchestrator: user_transactions remains the
		// primary stream; crypto-transactions and withdrawal-requests
		// are tried each cycle. Default both to empty so per-test
		// fixtures only need to mock the source they exercise.
		m.EXPECT().
			GetCryptoTransactions(gomock.Any(), gomock.Any()).
			Return(client.CryptoTransactionsResponse{}, nil).
			AnyTimes()
		m.EXPECT().
			GetWithdrawalRequests(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, nil).
			AnyTimes()
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
			Expect(err.Error()).To(ContainSubstring("test error"))
			Expect(err.Error()).To(ContainSubstring("user_transactions"))
			// Short-circuit on first source error: the engine drops the
			// response on any plugin error, so partial success cannot
			// be propagated. Next cycle retries from the unchanged
			// watermark.
			Expect(resp.Payments).To(BeEmpty())
			Expect(resp.NewState).To(BeEmpty())
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

		It("should map a sub-account transfer (positive amount) as PAYIN with transfer_pair metadata", func(ctx SpecContext) {
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
			Expect(payment.Type).To(Equal(models.PAYMENT_TYPE_PAYIN))
			Expect(payment.Metadata).To(HaveKeyWithValue("com.bitstamp.spec/transfer_direction", "incoming"))
			Expect(payment.Metadata).To(HaveKey("com.bitstamp.spec/transfer_pair_id"))
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
			Expect(newState.UserTransactions.LastTransactionID).To(Equal(int64(12350)))
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

	Context("multi-source orchestration", func() {
		var emptyState = []byte(`{}`)

		It("unions user_transactions, crypto-transactions and withdrawal-requests in one cycle", func(ctx SpecContext) {
			// Override the AnyTimes default with explicit non-empty rows.
			ctrl.Finish()
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m

			m.EXPECT().
				GetUserTransactions(gomock.Any(), gomock.Nil(), 100).
				Return([]client.UserTransaction{
					{ID: 1, Datetime: "2024-01-15 10:30:00.000000", Type: "0", CurrencyAmounts: map[string]string{"usd": "100.00"}},
				}, nil)
			m.EXPECT().
				GetCryptoTransactions(gomock.Any(), gomock.Any()).
				Return(client.CryptoTransactionsResponse{
					Deposits: []client.CryptoDeposit{{
						ID: 99, Currency: "BTC", TxID: "tx-99", Amount: json.Number("0.5"),
						Datetime: 1759995000, Status: "COMPLETED", Network: "bitcoin",
					}},
				}, nil)
			m.EXPECT().
				GetWithdrawalRequests(gomock.Any(), 1000, 0).
				Return([]client.WithdrawalRequest{{
					ID: 42, Datetime: "2025-09-25 14:42:59", Type: 0, Currency: "EUR", Amount: "100.00", Status: 2,
				}}, nil)

			resp, err := plg.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{State: emptyState, PageSize: 100})
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(3))
			refs := map[string]bool{}
			for _, p := range resp.Payments {
				refs[p.Reference] = true
			}
			Expect(refs["1"]).To(BeTrue(), "user_transactions row")
			Expect(refs["ct-dep:99"]).To(BeTrue(), "crypto deposit")
			Expect(refs["wr:42"]).To(BeTrue(), "fiat withdrawal request")
		})

		It("marks crypto-transactions as skipped when the PSP returns DerivativesUnsupportedError", func(ctx SpecContext) {
			ctrl.Finish()
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m

			// Default empties for user_transactions and withdrawal_requests.
			m.EXPECT().GetUserTransactions(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, nil).AnyTimes()
			m.EXPECT().GetWithdrawalRequests(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, nil).AnyTimes()

			// First cycle: crypto-transactions returns the typed error.
			m.EXPECT().GetCryptoTransactions(gomock.Any(), gomock.Any()).
				Return(client.CryptoTransactionsResponse{},
					&client.DerivativesUnsupportedError{Endpoint: "/api/v2/crypto-transactions/", Message: "Trade account does not support"})

			_, err := plg.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{State: emptyState, PageSize: 100})
			Expect(err).To(BeNil(), "typed derivatives error must not bubble up — orchestrator swallows + caches")

			// Second cycle: must NOT call GetCryptoTransactions again.
			// If it did, gomock would fail with "unexpected call" because
			// we set exactly one expectation above.
			_, err = plg.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{State: emptyState, PageSize: 100})
			Expect(err).To(BeNil())
		})

		It("emits the full crypto-transactions union (deposits + withdrawals + IOUs) and advances per-bucket watermarks", func(ctx SpecContext) {
			ctrl.Finish()
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m

			m.EXPECT().GetUserTransactions(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
			m.EXPECT().GetWithdrawalRequests(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
			m.EXPECT().GetCryptoTransactions(gomock.Any(), gomock.Any()).Return(client.CryptoTransactionsResponse{
				Deposits: []client.CryptoDeposit{
					{ID: 1, Currency: "BTC", TxID: "tx-d1", Amount: json.Number("0.1"), Datetime: 1000, Status: "COMPLETED", Network: "bitcoin"},
					{ID: 2, Currency: "BTC", TxID: "tx-d2", Amount: json.Number("0.2"), Datetime: 2000, Status: "COMPLETED", Network: "bitcoin"},
				},
				Withdrawals: []client.CryptoWithdrawal{
					{Currency: "BTC", TxID: "tx-w1", Amount: json.Number("0.05"), Datetime: 1500, Network: "bitcoin"},
				},
				RippleIOUTransactions: []client.RippleIOUTransaction{
					{Currency: "BTC", TxID: "tx-i1", Amount: json.Number("0.01"), Datetime: 800, Network: "bitcoin"},
				},
			}, nil)

			resp, err := plg.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{State: emptyState, PageSize: 100})
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(4), "2 deposits + 1 withdrawal + 1 ripple IOU")

			// Per-bucket watermarks must advance to the max datetime seen.
			var newState paymentsState
			Expect(json.Unmarshal(resp.NewState, &newState)).To(Succeed())
			Expect(newState.CryptoTransactions.DepositsSinceTs).To(Equal(int64(2000)))
			Expect(newState.CryptoTransactions.WithdrawalsSinceTs).To(Equal(int64(1500)))
			Expect(newState.CryptoTransactions.RipplesSinceTs).To(Equal(int64(800)))
		})

		It("withdrawal-requests filters by LastID watermark across cycles", func(ctx SpecContext) {
			ctrl.Finish()
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m

			// State already has LastID=10 — rows with id <= 10 must be skipped.
			startState := []byte(`{"withdrawalRequests": {"lastID": 10}}`)

			m.EXPECT().GetUserTransactions(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
			m.EXPECT().GetCryptoTransactions(gomock.Any(), gomock.Any()).Return(client.CryptoTransactionsResponse{}, nil)
			m.EXPECT().GetWithdrawalRequests(gomock.Any(), 1000, 0).Return([]client.WithdrawalRequest{
				{ID: 8, Datetime: "2025-09-25 14:42:59", Type: 0, Currency: "EUR", Amount: "10", Status: 2}, // skipped
				{ID: 11, Datetime: "2025-09-25 14:42:59", Type: 0, Currency: "EUR", Amount: "20", Status: 2}, // emitted
				{ID: 12, Datetime: "2025-09-25 14:42:59", Type: 0, Currency: "EUR", Amount: "30", Status: 2}, // emitted
			}, nil)

			resp, err := plg.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{State: startState, PageSize: 100})
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(2), "id=8 below watermark must be filtered out")
			refs := []string{resp.Payments[0].Reference, resp.Payments[1].Reference}
			Expect(refs).To(ConsistOf("wr:11", "wr:12"))

			var newState paymentsState
			Expect(json.Unmarshal(resp.NewState, &newState)).To(Succeed())
			Expect(newState.WithdrawalRequests.LastID).To(Equal(int64(12)))
		})

		It("fails the cycle on any source error and does not advance state", func(ctx SpecContext) {
			ctrl.Finish()
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m

			// user_transactions returns one row; crypto-transactions errors.
			// Per the engine contract (workflow drops the response on any
			// activity error), the partial success would be silently lost
			// anyway — we short-circuit explicitly so the cause is loud.
			m.EXPECT().GetUserTransactions(gomock.Any(), gomock.Any(), gomock.Any()).
				Return([]client.UserTransaction{
					{ID: 1, Datetime: "2024-01-15 10:30:00.000000", Type: "0", CurrencyAmounts: map[string]string{"usd": "100.00"}},
				}, nil)
			m.EXPECT().GetCryptoTransactions(gomock.Any(), gomock.Any()).
				Return(client.CryptoTransactionsResponse{}, errors.New("crypto-tx down"))
			// withdrawal-requests is NOT called — short-circuit on the first failing source.

			resp, err := plg.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{State: emptyState, PageSize: 100})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("crypto-tx down"))
			Expect(err.Error()).To(ContainSubstring("crypto_transactions"))
			Expect(resp.Payments).To(BeEmpty(), "cycle aborted — no partial response")
			Expect(resp.NewState).To(BeEmpty(), "no state advance on cycle failure")
		})
	})
})

func ptrInt64(v int64) *int64 {
	return &v
}
