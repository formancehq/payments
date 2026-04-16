package coinbaseprime

import (
	"errors"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/coinbaseprime/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
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
			logger: logging.NewDefaultLogger(GinkgoWriter, true, false, false),
			currencies: map[string]int{
				"USD":  2,
				"EUR":  2,
				"GBP":  2,
				"BTC":  8,
				"ETH":  18,
				"USDC": 6,
				"SOL":  9,
			},
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
						Type:    "PAYMENT_METHOD",
						Value:   "pm-123",
						Address: "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
					},
					TransferTo: &client.TransferEndpoint{
						Type:    "WALLET",
						Value:   "wallet1",
						Address: "bc1q...",
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
			// tx3 is a CONVERSION and is skipped by transactionToPayment
			Expect(resp.Payments).To(HaveLen(3))
			Expect(resp.HasMore).To(BeTrue())

			// Verify first payment (BTC deposit - completed, with transfer endpoints)
			Expect(resp.Payments[0].Reference).To(Equal("tx1"))
			Expect(resp.Payments[0].Type).To(Equal(models.PAYMENT_TYPE_PAYIN))
			Expect(resp.Payments[0].Status).To(Equal(models.PAYMENT_STATUS_SUCCEEDED))
			Expect(resp.Payments[0].Asset).To(Equal("BTC/8"))
			// 1.5 BTC = 150000000 (1.5 * 10^8)
			Expect(resp.Payments[0].Amount.Cmp(big.NewInt(150000000))).To(Equal(0))
			Expect(resp.Payments[0].SourceAccountReference).To(BeNil())
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
			Expect(resp.Payments[1].DestinationAccountReference).To(BeNil())

			// Verify third payment (ETH deposit - pending, 18 decimals, walletID fallback)
			// tx3 (CONVERSION) was skipped, so tx4 is at index 2
			Expect(resp.Payments[2].Reference).To(Equal("tx4"))
			Expect(resp.Payments[2].Type).To(Equal(models.PAYMENT_TYPE_PAYIN))
			Expect(resp.Payments[2].Status).To(Equal(models.PAYMENT_STATUS_PENDING))
			Expect(resp.Payments[2].Asset).To(Equal("ETH/18"))
			// 1.000000000000000001 ETH = 1000000000000000001 (1.000000000000000001 * 10^18)
			Expect(resp.Payments[2].Amount.Cmp(big.NewInt(1000000000000000001))).To(Equal(0))
			Expect(resp.Payments[2].SourceAccountReference).To(BeNil())
			Expect(resp.Payments[2].DestinationAccountReference).ToNot(BeNil())
			Expect(*resp.Payments[2].DestinationAccountReference).To(Equal("wallet4"))
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

		It("should populate metadata correctly", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10).Return(
				&client.TransactionsResponse{
					Transactions: sampleTransactions[:1],
					Pagination:   client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))

			md := resp.Payments[0].Metadata
			Expect(md[MetadataPrefix+"wallet_id"]).To(Equal("wallet1"))
			Expect(md[MetadataPrefix+"portfolio_id"]).To(Equal("portfolio1"))
			Expect(md[MetadataPrefix+"type"]).To(Equal("DEPOSIT"))
			Expect(md[MetadataPrefix+"status"]).To(Equal("TRANSACTION_DONE"))
			Expect(md[MetadataPrefix+"fees"]).To(Equal("0.001"))
			Expect(md[MetadataPrefix+"fee_symbol"]).To(Equal("BTC"))
			Expect(md[MetadataPrefix+"network"]).To(Equal("bitcoin"))
			Expect(md[MetadataPrefix+"blockchain_ids"]).To(Equal("0xabc123"))
			Expect(md).To(HaveKey(MetadataPrefix + "completed_at"))
			Expect(md[MetadataPrefix+"source_address"]).To(Equal("1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa"))
			Expect(md[MetadataPrefix+"deposit_address"]).To(Equal("bc1q..."))
		})

		It("should handle STAKE as transfer", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10).Return(
				&client.TransactionsResponse{
					Transactions: []client.Transaction{
						{
							ID:        "tx-stake",
							Type:      "STAKE",
							Status:    "TRANSACTION_DONE",
							CreatedAt: now,
							Amount:    "10.0",
							Symbol:    "ETH",
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			Expect(resp.Payments[0].Type).To(Equal(models.PAYMENT_TYPE_TRANSFER))
		})

		It("should handle UNSTAKE as transfer", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10).Return(
				&client.TransactionsResponse{
					Transactions: []client.Transaction{
						{
							ID:        "tx-unstake",
							Type:      "UNSTAKE",
							Status:    "TRANSACTION_DONE",
							CreatedAt: now,
							Amount:    "5.0",
							Symbol:    "ETH",
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			Expect(resp.Payments[0].Type).To(Equal(models.PAYMENT_TYPE_TRANSFER))
		})

		It("should handle DELEGATION as other", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10).Return(
				&client.TransactionsResponse{
					Transactions: []client.Transaction{
						{
							ID:        "tx-delegation",
							Type:      "DELEGATION",
							Status:    "TRANSACTION_DONE",
							CreatedAt: now,
							Amount:    "100.0",
							Symbol:    "SOL",
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			Expect(resp.Payments[0].Type).To(Equal(models.PaymentType(models.PAYMENT_TYPE_OTHER)))
		})

		It("should use dynamic currencies for previously unknown assets", func(ctx SpecContext) {
			// Add a dynamic currency that wouldn't be in the old static list
			plg.currencies["NEWTOKEN"] = 12

			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10).Return(
				&client.TransactionsResponse{
					Transactions: []client.Transaction{
						{
							ID:        "tx-new",
							Type:      "DEPOSIT",
							Status:    "TRANSACTION_DONE",
							CreatedAt: now,
							Amount:    "100.5",
							Symbol:    "NEWTOKEN",
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			Expect(resp.Payments[0].Asset).To(Equal("NEWTOKEN/12"))
		})

		It("should set account references to nil for non-WALLET types", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10).Return(
				&client.TransactionsResponse{
					Transactions: []client.Transaction{
						{
							ID:          "tx-payment-method",
							WalletID:    "",
							PortfolioID: "portfolio1",
							Type:        "DEPOSIT",
							Status:      "TRANSACTION_DONE",
							Symbol:      "BTC",
							Amount:      "1.0",
							CreatedAt:   now,
							TransferFrom: &client.TransferEndpoint{
								Type:  "PAYMENT_METHOD",
								Value: "pm-external-123",
							},
							TransferTo: &client.TransferEndpoint{
								Type:  "PAYMENT_METHOD",
								Value: "pm-external-456",
							},
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			Expect(resp.Payments[0].SourceAccountReference).To(BeNil())
			Expect(resp.Payments[0].DestinationAccountReference).To(BeNil())
		})

		It("should set account references for WALLET type but not for PAYMENT_METHOD type", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10).Return(
				&client.TransactionsResponse{
					Transactions: []client.Transaction{
						{
							ID:          "tx-mixed",
							WalletID:    "",
							PortfolioID: "portfolio1",
							Type:        "DEPOSIT",
							Status:      "TRANSACTION_DONE",
							Symbol:      "BTC",
							Amount:      "1.0",
							CreatedAt:   now,
							TransferFrom: &client.TransferEndpoint{
								Type:  "PAYMENT_METHOD",
								Value: "pm-external-123",
							},
							TransferTo: &client.TransferEndpoint{
								Type:  "WALLET",
								Value: "wallet-123",
							},
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			Expect(resp.Payments[0].SourceAccountReference).To(BeNil())
			Expect(resp.Payments[0].DestinationAccountReference).ToNot(BeNil())
			Expect(*resp.Payments[0].DestinationAccountReference).To(Equal("wallet-123"))
		})

		It("should handle case-insensitive WALLET type", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10).Return(
				&client.TransactionsResponse{
					Transactions: []client.Transaction{
						{
							ID:          "tx-lowercase",
							WalletID:    "",
							PortfolioID: "portfolio1",
							Type:        "DEPOSIT",
							Status:      "TRANSACTION_DONE",
							Symbol:      "BTC",
							Amount:      "1.0",
							CreatedAt:   now,
							TransferFrom: &client.TransferEndpoint{
								Type:  "wallet",
								Value: "wallet-lower",
							},
							TransferTo: &client.TransferEndpoint{
								Type:  "Wallet",
								Value: "wallet-mixed",
							},
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			Expect(resp.Payments[0].SourceAccountReference).ToNot(BeNil())
			Expect(*resp.Payments[0].SourceAccountReference).To(Equal("wallet-lower"))
			Expect(resp.Payments[0].DestinationAccountReference).ToNot(BeNil())
			Expect(*resp.Payments[0].DestinationAccountReference).To(Equal("wallet-mixed"))
		})

		It("should not include fee_symbol when fees and network_fees are zero", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10).Return(
				&client.TransactionsResponse{
					Transactions: []client.Transaction{
						{
							ID:          "tx-no-fees",
							WalletID:    "wallet1",
							PortfolioID: "portfolio1",
							Type:        "DEPOSIT",
							Status:      "TRANSACTION_DONE",
							Symbol:      "BTC",
							Amount:      "1.0",
							Fees:        "0",
							FeeSymbol:   "BTC",
							NetworkFees: "0",
							CreatedAt:   now,
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			md := resp.Payments[0].Metadata
			Expect(md).ToNot(HaveKey(MetadataPrefix + "fee_symbol"))
			Expect(md).ToNot(HaveKey(MetadataPrefix + "fees"))
			Expect(md).ToNot(HaveKey(MetadataPrefix + "network_fees"))
		})

		It("should include fee_symbol when fees is greater than zero", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10).Return(
				&client.TransactionsResponse{
					Transactions: []client.Transaction{
						{
							ID:          "tx-with-fees",
							WalletID:    "wallet1",
							PortfolioID: "portfolio1",
							Type:        "DEPOSIT",
							Status:      "TRANSACTION_DONE",
							Symbol:      "BTC",
							Amount:      "1.0",
							Fees:        "0.001",
							FeeSymbol:   "BTC",
							NetworkFees: "0",
							CreatedAt:   now,
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			md := resp.Payments[0].Metadata
			Expect(md[MetadataPrefix+"fee_symbol"]).To(Equal("BTC"))
			Expect(md[MetadataPrefix+"fees"]).To(Equal("0.001"))
		})

		It("should include fee_symbol when network_fees is greater than zero", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10).Return(
				&client.TransactionsResponse{
					Transactions: []client.Transaction{
						{
							ID:          "tx-with-network-fees",
							WalletID:    "wallet1",
							PortfolioID: "portfolio1",
							Type:        "DEPOSIT",
							Status:      "TRANSACTION_DONE",
							Symbol:      "ETH",
							Amount:      "1.0",
							Fees:        "0",
							FeeSymbol:   "ETH",
							NetworkFees: "0.0005",
							CreatedAt:   now,
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			md := resp.Payments[0].Metadata
			Expect(md[MetadataPrefix+"fee_symbol"]).To(Equal("ETH"))
			Expect(md[MetadataPrefix+"network_fees"]).To(Equal("0.0005"))
		})

		It("should include fee_symbol when either fees or network_fees is greater than zero", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10).Return(
				&client.TransactionsResponse{
					Transactions: []client.Transaction{
						{
							ID:          "tx-with-both-fees",
							WalletID:    "wallet1",
							PortfolioID: "portfolio1",
							Type:        "WITHDRAWAL",
							Status:      "TRANSACTION_DONE",
							Symbol:      "USDC",
							Amount:      "100.0",
							Fees:        "0.5",
							FeeSymbol:   "USDC",
							NetworkFees: "0.1",
							CreatedAt:   now,
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			md := resp.Payments[0].Metadata
			Expect(md[MetadataPrefix+"fee_symbol"]).To(Equal("USDC"))
			Expect(md[MetadataPrefix+"fees"]).To(Equal("0.5"))
			Expect(md[MetadataPrefix+"network_fees"]).To(Equal("0.1"))
		})
	})
})
