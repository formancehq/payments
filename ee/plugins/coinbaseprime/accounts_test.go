package coinbaseprime

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/coinbaseprime/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Coinbase Plugin Accounts", func() {
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
			},
			assetsLastSync: time.Now(),
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetching next accounts", func() {
		var (
			now           time.Time
			sampleWallets []client.Wallet
		)

		BeforeEach(func() {
			now = time.Now().UTC()

			sampleWallets = []client.Wallet{
				{
					ID:        "wallet1",
					Name:      "BTC Trading Wallet",
					Symbol:    "BTC",
					Type:      "TRADING",
					CreatedAt: now.Add(-48 * time.Hour),
				},
				{
					ID:        "wallet2",
					Name:      "USD Wallet",
					Symbol:    "USD",
					Type:      "TRADING",
					CreatedAt: now.Add(-24 * time.Hour),
				},
				{
					ID:        "wallet3",
					Name:      "ETH Vault",
					Symbol:    "ETH",
					Type:      "VAULT",
					CreatedAt: now,
				},
			}
		})

		It("should return an error - get wallets error", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetWallets(gomock.Any(), "TRADING", "", 10).Return(
				nil,
				errors.New("test error"),
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextAccountsResponse{}))
		})

		It("should fetch wallets successfully on the first type", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetWallets(gomock.Any(), "TRADING", "", 10).Return(
				&client.WalletsResponse{
					Wallets: sampleWallets,
					Pagination: client.Pagination{
						NextCursor: "cursor123",
						HasNext:    true,
					},
				},
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(3))
			Expect(resp.HasMore).To(BeTrue())

			Expect(resp.Accounts[0].Reference).To(Equal("wallet1"))
			Expect(*resp.Accounts[0].DefaultAsset).To(Equal("BTC/8"))
			Expect(resp.Accounts[0].Metadata["wallet_type"]).To(Equal("TRADING"))

			Expect(resp.Accounts[1].Reference).To(Equal("wallet2"))
			Expect(*resp.Accounts[1].DefaultAsset).To(Equal("USD/2"))

			Expect(resp.Accounts[2].Reference).To(Equal("wallet3"))
			Expect(*resp.Accounts[2].DefaultAsset).To(Equal("ETH/18"))
			Expect(resp.Accounts[2].Metadata["wallet_type"]).To(Equal("VAULT"))

			var newState accountsState
			Expect(json.Unmarshal(resp.NewState, &newState)).To(Succeed())
			Expect(newState.CurrentType).To(Equal("TRADING"))
			Expect(newState.Cursors["TRADING"]).To(Equal("cursor123"))
		})

		It("should skip unsupported currencies", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			unsupportedWallets := []client.Wallet{
				{
					ID:        "wallet1",
					Name:      "Unknown Wallet",
					Symbol:    "UNKNOWN_CURRENCY",
					Type:      "TRADING",
					CreatedAt: now,
				},
				{
					ID:        "wallet2",
					Name:      "BTC Wallet",
					Symbol:    "BTC",
					Type:      "TRADING",
					CreatedAt: now,
				},
			}

			m.EXPECT().GetWallets(gomock.Any(), "TRADING", "", 10).Return(
				&client.WalletsResponse{
					Wallets: unsupportedWallets,
					Pagination: client.Pagination{
						HasNext: false,
					},
				},
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(1))
			Expect(*resp.Accounts[0].DefaultAsset).To(Equal("BTC/8"))
		})

		It("should accept lowercase symbols", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			lowercaseWallets := []client.Wallet{
				{
					ID:        "wallet1",
					Name:      "btc wallet",
					Symbol:    "btc",
					Type:      "TRADING",
					CreatedAt: now,
				},
			}

			m.EXPECT().GetWallets(gomock.Any(), "TRADING", "", 10).Return(
				&client.WalletsResponse{
					Wallets: lowercaseWallets,
					Pagination: client.Pagination{
						HasNext: false,
					},
				},
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(1))
			Expect(*resp.Accounts[0].DefaultAsset).To(Equal("BTC/8"))
		})

		It("should resume mid-type from the persisted cursor and currentType", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{"currentType":"VAULT","cursors":{"VAULT":"existing-cursor"}}`),
				PageSize: 10,
			}

			m.EXPECT().GetWallets(gomock.Any(), "VAULT", "existing-cursor", 10).Return(
				&client.WalletsResponse{
					Wallets: []client.Wallet{},
					Pagination: client.Pagination{
						HasNext: false,
					},
				},
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(0))
			// VAULT done -> ONCHAIN next; framework keeps calling within this cycle.
			Expect(resp.HasMore).To(BeTrue())

			var newState accountsState
			Expect(json.Unmarshal(resp.NewState, &newState)).To(Succeed())
			Expect(newState.CurrentType).To(Equal("ONCHAIN"))
			// End-of-pagination with empty next_cursor preserves the prior cursor.
			Expect(newState.Cursors["VAULT"]).To(Equal("existing-cursor"))
		})

		It("should advance CurrentType when the current type finishes with a new cursor", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetWallets(gomock.Any(), "TRADING", "", 10).Return(
				&client.WalletsResponse{
					Wallets: nil,
					Pagination: client.Pagination{
						NextCursor: "last-trading-cursor",
						HasNext:    false,
					},
				},
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.HasMore).To(BeTrue())

			var state accountsState
			Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
			Expect(state.CurrentType).To(Equal("VAULT"))
			Expect(state.Cursors["TRADING"]).To(Equal("last-trading-cursor"))
		})

		It("should clear CurrentType when the final type finishes", func(ctx SpecContext) {
			prevState := accountsState{
				CurrentType: "WALLET_TYPE_OTHER",
				Cursors:     map[string]string{"TRADING": "t", "VAULT": "v"},
			}
			stateBytes, err := json.Marshal(prevState)
			Expect(err).To(BeNil())

			req := models.FetchNextAccountsRequest{
				State:    stateBytes,
				PageSize: 10,
			}

			m.EXPECT().GetWallets(gomock.Any(), "WALLET_TYPE_OTHER", "", 10).Return(
				&client.WalletsResponse{
					Wallets: nil,
					Pagination: client.Pagination{
						HasNext: false,
					},
				},
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.HasMore).To(BeFalse())

			var finalState accountsState
			Expect(json.Unmarshal(resp.NewState, &finalState)).To(Succeed())
			Expect(finalState.CurrentType).To(Equal(""))
			Expect(finalState.Cursors["TRADING"]).To(Equal("t"))
			Expect(finalState.Cursors["VAULT"]).To(Equal("v"))
		})
	})

	Describe("state helpers", func() {
		It("resolveTypeIndex falls back to 0 for unknown types", func() {
			Expect(resolveTypeIndex("")).To(Equal(0))
			Expect(resolveTypeIndex("REMOVED_TYPE")).To(Equal(0))
			Expect(resolveTypeIndex("TRADING")).To(Equal(0))
			Expect(resolveTypeIndex("ONCHAIN")).To(Equal(2))
			Expect(resolveTypeIndex("WALLET_TYPE_OTHER")).To(Equal(4))
		})

		It("walletTypes is the expected ordered list", func() {
			Expect(walletTypes).To(Equal([]string{
				"TRADING",
				"VAULT",
				"ONCHAIN",
				"QC",
				"WALLET_TYPE_OTHER",
			}))
		})
	})
})
