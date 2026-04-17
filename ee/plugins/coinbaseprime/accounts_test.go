package coinbaseprime

import (
	"errors"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/ee/plugins/coinbaseprime/client"
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

			m.EXPECT().GetWallets(gomock.Any(), "", 10).Return(
				nil,
				errors.New("test error"),
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextAccountsResponse{}))
		})

		It("should fetch wallets successfully", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetWallets(gomock.Any(), "", 10).Return(
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

			// Verify BTC wallet
			Expect(resp.Accounts[0].Reference).To(Equal("wallet1"))
			Expect(resp.Accounts[0].CreatedAt).To(Equal(sampleWallets[0].CreatedAt))
			Expect(*resp.Accounts[0].DefaultAsset).To(Equal("BTC/8"))
			Expect(*resp.Accounts[0].Name).To(Equal("BTC Trading Wallet"))
			Expect(resp.Accounts[0].Metadata["wallet_type"]).To(Equal("TRADING"))

			// Verify USD wallet
			Expect(resp.Accounts[1].Reference).To(Equal("wallet2"))
			Expect(resp.Accounts[1].CreatedAt).To(Equal(sampleWallets[1].CreatedAt))
			Expect(*resp.Accounts[1].DefaultAsset).To(Equal("USD/2"))

			// Verify ETH wallet
			Expect(resp.Accounts[2].Reference).To(Equal("wallet3"))
			Expect(resp.Accounts[2].CreatedAt).To(Equal(sampleWallets[2].CreatedAt))
			Expect(*resp.Accounts[2].DefaultAsset).To(Equal("ETH/18"))
			Expect(resp.Accounts[2].Metadata["wallet_type"]).To(Equal("VAULT"))
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

			m.EXPECT().GetWallets(gomock.Any(), "", 10).Return(
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

			m.EXPECT().GetWallets(gomock.Any(), "", 10).Return(
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

		It("should use cursor for pagination", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{"cursor": "existing-cursor"}`),
				PageSize: 10,
			}

			m.EXPECT().GetWallets(gomock.Any(), "existing-cursor", 10).Return(
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
			Expect(resp.HasMore).To(BeFalse())
		})

		It("merges returned wallets into p.wallets as a side effect", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetWallets(gomock.Any(), "", 10).Return(
				&client.WalletsResponse{
					Wallets: []client.Wallet{
						{ID: "wallet-btc", Symbol: "BTC"},
						{ID: "wallet-usd", Symbol: "USD"},
						{ID: "wallet-eth", Symbol: "ETH"},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			_, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(plg.wallets).To(HaveKeyWithValue("BTC", "wallet-btc"))
			Expect(plg.wallets).To(HaveKeyWithValue("USD", "wallet-usd"))
			Expect(plg.wallets).To(HaveKeyWithValue("ETH", "wallet-eth"))
		})

		It("merges across successive pagination calls without clearing prior entries", func(ctx SpecContext) {
			// Seed existing entries as if Install had populated them.
			plg.wallets = map[string]string{
				"BTC": "wallet-btc-original",
				"EUR": "wallet-eur-original",
			}

			m.EXPECT().GetWallets(gomock.Any(), "", 10).Return(
				&client.WalletsResponse{
					Wallets: []client.Wallet{
						{ID: "wallet-usd", Symbol: "USD"},
						{ID: "wallet-eth", Symbol: "ETH"},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}
			_, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			// New entries added.
			Expect(plg.wallets).To(HaveKeyWithValue("USD", "wallet-usd"))
			Expect(plg.wallets).To(HaveKeyWithValue("ETH", "wallet-eth"))
			// Original entries preserved (merge-only, never clear).
			Expect(plg.wallets).To(HaveKeyWithValue("BTC", "wallet-btc-original"))
			Expect(plg.wallets).To(HaveKeyWithValue("EUR", "wallet-eur-original"))
		})

		It("updates wallet IDs when the same symbol reappears with a different ID", func(ctx SpecContext) {
			plg.wallets = map[string]string{
				"BTC": "wallet-btc-old",
			}

			m.EXPECT().GetWallets(gomock.Any(), "", 10).Return(
				&client.WalletsResponse{
					Wallets: []client.Wallet{
						{ID: "wallet-btc-new", Symbol: "BTC"},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}
			_, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(plg.wallets).To(HaveKeyWithValue("BTC", "wallet-btc-new"))
		})

		It("records wallet IDs even for currencies filtered from the response", func(ctx SpecContext) {
			// UNKNOWN is not in p.currencies, so the wallet is skipped from the
			// returned accounts slice. But its ID should still be merged into
			// p.wallets so it becomes resolvable after a later assets refresh.
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetWallets(gomock.Any(), "", 10).Return(
				&client.WalletsResponse{
					Wallets: []client.Wallet{
						{ID: "wallet-unknown", Symbol: "UNKNOWN"},
					},
					Pagination: client.Pagination{HasNext: false},
				},
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(0))
			Expect(plg.wallets).To(HaveKeyWithValue("UNKNOWN", "wallet-unknown"))
		})
	})
})
