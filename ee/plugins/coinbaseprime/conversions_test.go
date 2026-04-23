package coinbaseprime

import (
	"encoding/json"
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

var _ = Describe("Coinbase Plugin Conversions", func() {
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
				"USDC": 6,
				"EUR":  2,
				"BTC":  8,
			},
			networkSymbols: map[string]string{},
			assetsLastSync: time.Now(),
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetching next conversions", func() {
		It("should return an error - GetTransactions error", func(ctx SpecContext) {
			req := models.FetchNextConversionsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10, TransactionTypeConversion).Return(
				nil,
				errors.New("test error"),
			)

			resp, err := plg.FetchNextConversions(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(ContainSubstring("test error")))
			Expect(resp).To(Equal(models.FetchNextConversionsResponse{}))
		})

		It("should skip non-CONVERSION transactions", func(ctx SpecContext) {
			req := models.FetchNextConversionsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10, TransactionTypeConversion).Return(
				&client.TransactionsResponse{
					Transactions: []client.Transaction{
						{
							ID:        "tx-deposit",
							Type:      "DEPOSIT",
							Symbol:    "USD",
							Amount:    "100",
							Status:    "TRANSACTION_DONE",
							CreatedAt: time.Date(2026, 2, 9, 15, 0, 0, 0, time.UTC),
						},
						{
							ID:        "tx-withdrawal",
							Type:      "WITHDRAWAL",
							Symbol:    "USDC",
							Amount:    "50",
							Status:    "TRANSACTION_DONE",
							CreatedAt: time.Date(2026, 2, 9, 16, 0, 0, 0, time.UTC),
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextConversions(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Conversions).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
		})

		It("should convert a USD to USDC conversion correctly", func(ctx SpecContext) {
			req := models.FetchNextConversionsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10, TransactionTypeConversion).Return(
				&client.TransactionsResponse{
					Transactions: []client.Transaction{
						{
							ID:                "55fa2a0b-5dd4-4924-bba8-214955ffd5dc",
							WalletID:          "570270d8-bbeb-54fa-a357-459200978943",
							PortfolioID:       "842695ec-67da-4227-a70f-105dbf2bd62a",
							Type:              TransactionTypeConversion,
							Status:            "TRANSACTION_DONE",
							Symbol:            "USD",
							DestinationSymbol: "USDC",
							Amount:            "100",
							Fees:              "0",
							FeeSymbol:         "USD",
							CreatedAt:         time.Date(2026, 2, 9, 15, 33, 25, 0, time.UTC),
							TransferFrom: &client.TransferEndpoint{
								Type:  "WALLET",
								Value: "570270d8-bbeb-54fa-a357-459200978943",
							},
							TransferTo: &client.TransferEndpoint{
								Type:  "WALLET",
								Value: "2fa3280b-4fc2-5cc6-8c68-6f8282b0b936",
							},
							TransactionID: "CA72CE50",
						},
					},
					Pagination: client.Pagination{
						NextCursor: "cursor-123",
						HasNext:    true,
					},
				},
				nil,
			)

			resp, err := plg.FetchNextConversions(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Conversions).To(HaveLen(1))
			Expect(resp.HasMore).To(BeTrue())

			conv := resp.Conversions[0]
			Expect(conv.Reference).To(Equal("55fa2a0b-5dd4-4924-bba8-214955ffd5dc"))
			Expect(conv.SourceAsset).To(Equal("USD/2"))
			Expect(conv.DestinationAsset).To(Equal("USDC/6"))

			// Source: 100 USD at precision 2 = 10000
			Expect(conv.SourceAmount).ToNot(BeNil())
			Expect(conv.SourceAmount.Cmp(big.NewInt(10000))).To(Equal(0))

			// Target: 100 USDC at precision 6 = 100000000
			Expect(conv.DestinationAmount).ToNot(BeNil())
			Expect(conv.DestinationAmount.Cmp(big.NewInt(100000000))).To(Equal(0))

			Expect(conv.Status).To(Equal(models.CONVERSION_STATUS_COMPLETED))
			Expect(conv.SourceAccountReference).ToNot(BeNil())
			Expect(*conv.SourceAccountReference).To(Equal("570270d8-bbeb-54fa-a357-459200978943"))
			Expect(conv.DestinationAccountReference).ToNot(BeNil())
			Expect(*conv.DestinationAccountReference).To(Equal("2fa3280b-4fc2-5cc6-8c68-6f8282b0b936"))

			// Fee is "0" so should be nil
			Expect(conv.Fee).To(BeNil())
			Expect(conv.FeeAsset).To(BeNil())

			// Metadata
			Expect(conv.Metadata[MetadataPrefix+"transaction_id"]).To(Equal("CA72CE50"))
			Expect(conv.Metadata[MetadataPrefix+"portfolio_id"]).To(Equal("842695ec-67da-4227-a70f-105dbf2bd62a"))

			// Pagination state
			var state incrementalState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.Cursor).To(Equal("cursor-123"))
		})

		It("should handle conversion with fees", func(ctx SpecContext) {
			req := models.FetchNextConversionsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10, TransactionTypeConversion).Return(
				&client.TransactionsResponse{
					Transactions: []client.Transaction{
						{
							ID:                "conv-with-fee",
							WalletID:          "wallet-usd",
							Type:              TransactionTypeConversion,
							Status:            "TRANSACTION_DONE",
							Symbol:            "USD",
							DestinationSymbol: "USDC",
							Amount:            "1000",
							Fees:              "0.50",
							FeeSymbol:         "USD",
							CreatedAt:         time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
							TransferFrom:      &client.TransferEndpoint{Value: "wallet-usd"},
							TransferTo:        &client.TransferEndpoint{Value: "wallet-usdc"},
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextConversions(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Conversions).To(HaveLen(1))

			conv := resp.Conversions[0]
			Expect(conv.Fee).ToNot(BeNil())
			Expect(conv.Fee.Cmp(big.NewInt(50))).To(Equal(0))
			Expect(conv.FeeAsset).ToNot(BeNil())
			Expect(*conv.FeeAsset).To(Equal("USD/2"))
		})

		It("should map pending status", func(ctx SpecContext) {
			req := models.FetchNextConversionsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10, TransactionTypeConversion).Return(
				&client.TransactionsResponse{
					Transactions: []client.Transaction{
						{
							ID:                "conv-pending",
							WalletID:          "wallet-usd",
							Type:              TransactionTypeConversion,
							Status:            "TRANSACTION_PENDING",
							Symbol:            "USD",
							DestinationSymbol: "USDC",
							Amount:            "50",
							CreatedAt:         time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextConversions(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Conversions[0].Status).To(Equal(models.CONVERSION_STATUS_PENDING))
		})

		It("should map failed status", func(ctx SpecContext) {
			req := models.FetchNextConversionsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10, TransactionTypeConversion).Return(
				&client.TransactionsResponse{
					Transactions: []client.Transaction{
						{
							ID:                "conv-failed",
							WalletID:          "wallet-usd",
							Type:              TransactionTypeConversion,
							Status:            "TRANSACTION_FAILED",
							Symbol:            "USD",
							DestinationSymbol: "USDC",
							Amount:            "50",
							CreatedAt:         time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextConversions(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Conversions[0].Status).To(Equal(models.CONVERSION_STATUS_FAILED))
		})

		It("should use cursor from state", func(ctx SpecContext) {
			req := models.FetchNextConversionsRequest{
				State:    []byte(`{"cursor":"existing-cursor"}`),
				PageSize: 25,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "existing-cursor", 25, TransactionTypeConversion).Return(
				&client.TransactionsResponse{
					Transactions: []client.Transaction{},
					Pagination:   client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextConversions(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Conversions).To(HaveLen(0))
		})

		It("should skip conversion with unsupported source currency", func(ctx SpecContext) {
			req := models.FetchNextConversionsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetTransactions(gomock.Any(), "", 10, TransactionTypeConversion).Return(
				&client.TransactionsResponse{
					Transactions: []client.Transaction{
						{
							ID:                "conv-unknown",
							WalletID:          "wallet-x",
							Type:              TransactionTypeConversion,
							Status:            "TRANSACTION_DONE",
							Symbol:            "UNKNOWNCOIN",
							DestinationSymbol: "USDC",
							Amount:            "100",
							CreatedAt:         time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
						},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextConversions(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Conversions).To(HaveLen(0))
		})
	})
})
