package coinbase

import (
	"errors"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/coinbase/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Coinbase Plugin Payments", func() {
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
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetching next payments", func() {
		var (
			now                time.Time
			completedAt        time.Time
			sampleTransactions []client.Transaction
		)

		BeforeEach(func() {
			now = time.Now().UTC()
			completedAt = now.Add(-time.Hour)

			sampleTransactions = []client.Transaction{
				{
					ID:          "tx1",
					WalletID:    "wallet1",
					PortfolioID: "portfolio1",
					Type:        "DEPOSIT",
					Status:      "TRANSACTION_DONE",
					Symbol:      "BTC",
					Amount:      "1.5",
					Fees:        "0.001",
					FeeSymbol:   "BTC",
					CreatedAt:   now.Add(-2 * time.Hour),
					CompletedAt: &completedAt,
					TransferFrom: &client.TransferEndpoint{
						Type:  "PAYMENT_METHOD",
						Value: "pm-123",
					},
					TransferTo: &client.TransferEndpoint{
						Type:  "WALLET",
						Value: "wallet1",
					},
					Network:       "bitcoin",
					BlockchainIDs: []string{"0xabc123"},
				},
				{
					ID:          "tx2",
					WalletID:    "wallet2",
					PortfolioID: "portfolio1",
					Type:        "WITHDRAWAL",
					Status:      "TRANSACTION_PENDING",
					Symbol:      "USD",
					Amount:      "500.00",
					CreatedAt:   now.Add(-time.Hour),
					TransferFrom: &client.TransferEndpoint{
						Type:  "WALLET",
						Value: "wallet2",
					},
					TransferTo: &client.TransferEndpoint{
						Type:  "PAYMENT_METHOD",
						Value: "external-bank",
					},
				},
				{
					ID:          "tx3",
					WalletID:    "wallet3",
					PortfolioID: "portfolio1",
					Type:        "CONVERSION",
					Status:      "TRANSACTION_DONE",
					Symbol:      "USDC",
					Amount:      "100.00",
					CreatedAt:   now,
					CompletedAt: &completedAt,
					TransferFrom: &client.TransferEndpoint{
						Type:  "WALLET",
						Value: "wallet-a",
					},
					TransferTo: &client.TransferEndpoint{
						Type:  "WALLET",
						Value: "wallet-b",
					},
				},
				{
					ID:          "tx4",
					WalletID:    "wallet4",
					PortfolioID: "portfolio1",
					Type:        "DEPOSIT",
					Status:      "TRANSACTION_IMPORT_PENDING",
					Symbol:      "ETH",
					Amount:      "1.000000000000000001",
					CreatedAt:   now,
				},
			}
		})

		It("should return an error - get transactions error", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10).Return(
				nil,
				errors.New("test error"),
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should fetch payments successfully", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10).Return(
				&client.TransactionsResponse{
					Transactions: sampleTransactions,
					Pagination: client.Pagination{
						NextCursor: "cursor123",
						HasNext:    true,
					},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(4))
			Expect(resp.HasMore).To(BeTrue())

			// Verify first payment (BTC deposit - completed, with transfer endpoints)
			Expect(resp.Payments[0].Reference).To(Equal("tx1"))
			Expect(resp.Payments[0].Type).To(Equal(models.PAYMENT_TYPE_PAYIN))
			Expect(resp.Payments[0].Status).To(Equal(models.PAYMENT_STATUS_SUCCEEDED))
			Expect(resp.Payments[0].Asset).To(Equal("BTC/8"))
			// 1.5 BTC = 150000000 (1.5 * 10^8)
			Expect(resp.Payments[0].Amount.Cmp(big.NewInt(150000000))).To(Equal(0))
			Expect(resp.Payments[0].SourceAccountReference).ToNot(BeNil())
			Expect(*resp.Payments[0].SourceAccountReference).To(Equal("pm-123"))
			Expect(resp.Payments[0].DestinationAccountReference).ToNot(BeNil())
			Expect(*resp.Payments[0].DestinationAccountReference).To(Equal("wallet1"))

			// Verify second payment (USD withdrawal - pending, with transfer endpoints)
			Expect(resp.Payments[1].Reference).To(Equal("tx2"))
			Expect(resp.Payments[1].Type).To(Equal(models.PAYMENT_TYPE_PAYOUT))
			Expect(resp.Payments[1].Status).To(Equal(models.PAYMENT_STATUS_PENDING))
			Expect(resp.Payments[1].Asset).To(Equal("USD/2"))
			// 500.00 USD = 50000 (500.00 * 10^2)
			Expect(resp.Payments[1].Amount.Cmp(big.NewInt(50000))).To(Equal(0))
			Expect(resp.Payments[1].SourceAccountReference).ToNot(BeNil())
			Expect(*resp.Payments[1].SourceAccountReference).To(Equal("wallet2"))
			Expect(resp.Payments[1].DestinationAccountReference).ToNot(BeNil())
			Expect(*resp.Payments[1].DestinationAccountReference).To(Equal("external-bank"))

			// Verify third payment (USDC conversion - completed)
			Expect(resp.Payments[2].Reference).To(Equal("tx3"))
			Expect(resp.Payments[2].Type).To(Equal(models.PAYMENT_TYPE_TRANSFER))
			Expect(resp.Payments[2].Status).To(Equal(models.PAYMENT_STATUS_SUCCEEDED))
			Expect(resp.Payments[2].Asset).To(Equal("USDC/6"))
			// 100.00 USDC = 100000000 (100.00 * 10^6)
			Expect(resp.Payments[2].Amount.Cmp(big.NewInt(100000000))).To(Equal(0))
			Expect(resp.Payments[2].SourceAccountReference).ToNot(BeNil())
			Expect(*resp.Payments[2].SourceAccountReference).To(Equal("wallet-a"))
			Expect(resp.Payments[2].DestinationAccountReference).ToNot(BeNil())
			Expect(*resp.Payments[2].DestinationAccountReference).To(Equal("wallet-b"))

			// Verify fourth payment (ETH deposit - pending, 18 decimals, walletID fallback)
			Expect(resp.Payments[3].Reference).To(Equal("tx4"))
			Expect(resp.Payments[3].Type).To(Equal(models.PAYMENT_TYPE_PAYIN))
			Expect(resp.Payments[3].Status).To(Equal(models.PAYMENT_STATUS_PENDING))
			Expect(resp.Payments[3].Asset).To(Equal("ETH/18"))
			// 1.000000000000000001 ETH = 1000000000000000001 (1.000000000000000001 * 10^18)
			Expect(resp.Payments[3].Amount.Cmp(big.NewInt(1000000000000000001))).To(Equal(0))
			Expect(resp.Payments[3].SourceAccountReference).To(BeNil())
			Expect(resp.Payments[3].DestinationAccountReference).ToNot(BeNil())
			Expect(*resp.Payments[3].DestinationAccountReference).To(Equal("wallet4"))
		})

		It("should skip unsupported currencies", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			unsupportedTransactions := []client.Transaction{
				{
					ID:        "tx1",
					Type:      "DEPOSIT",
					Status:    "TRANSACTION_DONE",
					CreatedAt: now,
					Amount:    "100",
					Symbol:    "UNKNOWN_CURRENCY",
				},
				{
					ID:        "tx2",
					Type:      "DEPOSIT",
					Status:    "TRANSACTION_DONE",
					CreatedAt: now,
					Amount:    "1.0",
					Symbol:    "BTC",
				},
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10).Return(
				&client.TransactionsResponse{
					Transactions: unsupportedTransactions,
					Pagination: client.Pagination{
						HasNext: false,
					},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			Expect(resp.Payments[0].Reference).To(Equal("tx2"))
		})

		It("should handle failed transactions", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			failedTransactions := []client.Transaction{
				{
					ID:        "tx1",
					Type:      "WITHDRAWAL",
					Status:    "TRANSACTION_FAILED",
					CreatedAt: now,
					Amount:    "100.00",
					Symbol:    "USD",
				},
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10).Return(
				&client.TransactionsResponse{
					Transactions: failedTransactions,
					Pagination: client.Pagination{
						HasNext: false,
					},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			Expect(resp.Payments[0].Status).To(Equal(models.PAYMENT_STATUS_FAILED))
		})

		It("should handle cancelled transactions", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10).Return(
				&client.TransactionsResponse{
					Transactions: []client.Transaction{
						{
							ID:        "tx-cancelled",
							Type:      "WITHDRAWAL",
							Status:    "TRANSACTION_CANCELLED",
							CreatedAt: now,
							Amount:    "100.00",
							Symbol:    "USD",
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			Expect(resp.Payments[0].Status).To(Equal(models.PAYMENT_STATUS_CANCELLED))
		})

		It("should handle expired transactions", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10).Return(
				&client.TransactionsResponse{
					Transactions: []client.Transaction{
						{
							ID:        "tx-expired",
							Type:      "DEPOSIT",
							Status:    "TRANSACTION_EXPIRED",
							CreatedAt: now,
							Amount:    "50.00",
							Symbol:    "USD",
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			Expect(resp.Payments[0].Status).To(Equal(models.PAYMENT_STATUS_EXPIRED))
		})

		It("should handle retried transactions as unknown", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10).Return(
				&client.TransactionsResponse{
					Transactions: []client.Transaction{
						{
							ID:        "tx-retried",
							Type:      "WITHDRAWAL",
							Status:    "TRANSACTION_RETRIED",
							CreatedAt: now,
							Amount:    "25.00",
							Symbol:    "USD",
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			Expect(resp.Payments[0].Status).To(Equal(models.PAYMENT_STATUS_UNKNOWN))
		})

		It("should accept lowercase symbols", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			lowercaseTransactions := []client.Transaction{
				{
					ID:        "tx-lower",
					Type:      "DEPOSIT",
					Status:    "TRANSACTION_DONE",
					CreatedAt: now,
					Amount:    "1.0",
					Symbol:    "btc",
				},
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10).Return(
				&client.TransactionsResponse{
					Transactions: lowercaseTransactions,
					Pagination: client.Pagination{
						HasNext: false,
					},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			Expect(resp.Payments[0].Asset).To(Equal("BTC/8"))
		})

		It("should use cursor for pagination", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{"cursor": "existing-cursor"}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "existing-cursor", 10).Return(
				&client.TransactionsResponse{
					Transactions: []client.Transaction{},
					Pagination: client.Pagination{
						HasNext: false,
					},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
		})
	})
})
