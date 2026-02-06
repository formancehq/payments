package coinbase

import (
	"errors"

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
		var sampleAccounts []client.Account

		BeforeEach(func() {
			sampleAccounts = []client.Account{
				{
					ID:             "acc1",
					Currency:       "BTC",
					Balance:        "1.5",
					Available:      "1.0",
					Hold:           "0.5",
					ProfileID:      "profile1",
					TradingEnabled: true,
				},
				{
					ID:             "acc2",
					Currency:       "USD",
					Balance:        "1000.00",
					Available:      "900.00",
					Hold:           "100.00",
					ProfileID:      "profile1",
					TradingEnabled: true,
				},
				{
					ID:             "acc3",
					Currency:       "ETH",
					Balance:        "10.5",
					Available:      "10.0",
					Hold:           "0.5",
					ProfileID:      "profile1",
					TradingEnabled: false,
				},
			}
		})

		It("should return an error - get accounts error", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State: []byte(`{}`),
			}

			m.EXPECT().GetAccounts(gomock.Any()).Return(
				nil,
				errors.New("test error"),
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextAccountsResponse{}))
		})

		It("should fetch accounts successfully", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State: []byte(`{}`),
			}

			m.EXPECT().GetAccounts(gomock.Any()).Return(
				sampleAccounts,
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(3))
			Expect(resp.HasMore).To(BeFalse())

			// Verify BTC account has correct default asset (8 decimals)
			Expect(*resp.Accounts[0].DefaultAsset).To(Equal("BTC/8"))
			Expect(*resp.Accounts[0].Name).To(Equal("BTC Wallet"))

			// Verify USD account has correct default asset (2 decimals)
			Expect(*resp.Accounts[1].DefaultAsset).To(Equal("USD/2"))
			Expect(*resp.Accounts[1].Name).To(Equal("USD Wallet"))

			// Verify ETH account has correct default asset (18 decimals)
			Expect(*resp.Accounts[2].DefaultAsset).To(Equal("ETH/18"))
		})

		It("should skip unsupported currencies", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State: []byte(`{}`),
			}

			unsupportedAccounts := []client.Account{
				{
					ID:       "acc1",
					Currency: "UNKNOWN_CURRENCY",
					Balance:  "100",
				},
				{
					ID:       "acc2",
					Currency: "BTC",
					Balance:  "1.0",
				},
			}

			m.EXPECT().GetAccounts(gomock.Any()).Return(
				unsupportedAccounts,
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			// Only BTC account should be returned
			Expect(resp.Accounts).To(HaveLen(1))
			Expect(*resp.Accounts[0].DefaultAsset).To(Equal("BTC/8"))
		})

		It("should not fetch again when already fetched", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State: []byte(`{"fetched": true}`),
			}

			// No mock expectation - GetAccounts should not be called

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
		})
	})
})
