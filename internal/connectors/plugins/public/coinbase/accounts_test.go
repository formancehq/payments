package coinbase

import (
	"errors"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/coinbase/client"
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
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetching next accounts", func() {
		var (
			now            time.Time
			sampleWallets  []client.Wallet
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
	})
})
