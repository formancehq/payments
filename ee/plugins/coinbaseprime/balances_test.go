package coinbaseprime

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/ee/plugins/coinbaseprime/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Coinbase Plugin Balances", func() {
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
			currencies: map[string]int{
				"USD":  2,
				"EUR":  2,
				"GBP":  2,
				"BTC":  8,
				"ETH":  18,
				"USDC": 6,
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetching next balances", func() {
		var sampleBalances []client.Balance

		BeforeEach(func() {
			sampleBalances = []client.Balance{
				{
					Symbol:             "BTC",
					Amount:             "1.5",
					Holds:              "0.5",
					WithdrawableAmount: "1.0",
					FiatAmount:         "75000.00",
				},
				{
					Symbol:             "USD",
					Amount:             "1000.50",
					Holds:              "100.50",
					WithdrawableAmount: "900.00",
					FiatAmount:         "1000.50",
				},
			}
		})

		It("should return an error - missing from payload", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				FromPayload: nil,
			}

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing from payload"))
			Expect(resp).To(Equal(models.FetchNextBalancesResponse{}))
		})

		It("should return an error - get balance for wallet error", func(ctx SpecContext) {
			fromPayload, _ := json.Marshal(models.PSPAccount{
				Reference: "wallet1",
			})
			req := models.FetchNextBalancesRequest{
				FromPayload: fromPayload,
				PageSize:    10,
			}

			m.EXPECT().GetBalanceForWallet(gomock.Any(), "wallet1").Return(
				nil,
				errors.New("test error"),
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextBalancesResponse{}))
		})

		It("should fetch balances with correct precision", func(ctx SpecContext) {
			fromPayload, _ := json.Marshal(models.PSPAccount{
				Reference: "wallet1",
			})
			req := models.FetchNextBalancesRequest{
				FromPayload: fromPayload,
				PageSize:    10,
			}

			m.EXPECT().GetBalanceForWallet(gomock.Any(), "wallet1").Return(
				&client.WalletBalanceResponse{Balance: sampleBalances[0]},
				nil,
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(1))
			Expect(resp.HasMore).To(BeFalse())

			// BTC has 8 decimals, so 1.5 BTC = 150000000 (1.5 * 10^8)
			Expect(resp.Balances[0].Asset).To(Equal("BTC/8"))
			Expect(resp.Balances[0].Amount.Cmp(big.NewInt(150000000))).To(Equal(0))
			Expect(resp.Balances[0].AccountReference).To(Equal("wallet1"))
		})

		It("should normalize lowercase balance symbols from API", func(ctx SpecContext) {
			fromPayload, _ := json.Marshal(models.PSPAccount{
				Reference: "wallet1",
			})
			req := models.FetchNextBalancesRequest{
				FromPayload: fromPayload,
				PageSize:    10,
			}

			m.EXPECT().GetBalanceForWallet(gomock.Any(), "wallet1").Return(
				&client.WalletBalanceResponse{
					Balance: client.Balance{
						Symbol: "btc",
						Amount: "1.5",
					},
				},
				nil,
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(1))
			Expect(resp.Balances[0].Asset).To(Equal("BTC/8"))
			Expect(resp.Balances[0].Amount.Cmp(big.NewInt(150000000))).To(Equal(0))
		})

		It("should parse amount with currency-specific decimals", func(ctx SpecContext) {
			fromPayload, _ := json.Marshal(models.PSPAccount{
				Reference: "wallet-eth",
			})
			req := models.FetchNextBalancesRequest{
				FromPayload: fromPayload,
				PageSize:    10,
			}

			m.EXPECT().GetBalanceForWallet(gomock.Any(), "wallet-eth").Return(
				&client.WalletBalanceResponse{
					Balance: client.Balance{
						Symbol: "ETH",
						Amount: "0.000000000000000001",
					},
				},
				nil,
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(1))

			Expect(resp.Balances[0].Asset).To(Equal("ETH/18"))
			Expect(resp.Balances[0].Amount.Cmp(big.NewInt(1))).To(Equal(0))
		})

		It("should return empty balances for unsupported currencies", func(ctx SpecContext) {
			fromPayload, _ := json.Marshal(models.PSPAccount{
				Reference: "wallet-btc",
			})
			req := models.FetchNextBalancesRequest{
				FromPayload: fromPayload,
				PageSize:    10,
			}

			m.EXPECT().GetBalanceForWallet(gomock.Any(), "wallet-btc").Return(
				&client.WalletBalanceResponse{
					Balance: client.Balance{
						Symbol: "UNKNOWN_CURRENCY",
						Amount: "100",
					},
				},
				nil,
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(0))
		})

		It("should return zero balance when wallet has no balance", func(ctx SpecContext) {
			fromPayload, _ := json.Marshal(models.PSPAccount{
				Reference: "wallet1",
			})
			req := models.FetchNextBalancesRequest{
				FromPayload: fromPayload,
				PageSize:    10,
			}

			m.EXPECT().GetBalanceForWallet(gomock.Any(), "wallet1").Return(
				&client.WalletBalanceResponse{
					Balance: client.Balance{
						Symbol: "BTC",
						Amount: "0",
					},
				},
				nil,
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(1))
			Expect(resp.Balances[0].Amount.Cmp(big.NewInt(0))).To(Equal(0))
		})
	})
})
