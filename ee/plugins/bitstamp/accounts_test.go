package bitstamp

import (
	"errors"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Bitstamp Plugin Accounts", func() {
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
		}
		plg.currLoaded.Store(true)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetching next accounts", func() {
		It("should return an error - get balances error", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetAccountBalances(gomock.Any()).Return(
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
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetAccountBalances(gomock.Any()).Return(
				[]client.AccountBalance{
					{Currency: "btc", Total: "1.50000000", Available: "1.00000000", Reserved: "0.50000000"},
					{Currency: "usd", Total: "5000.00", Available: "4500.00", Reserved: "500.00"},
					{Currency: "eth", Total: "10.000000000000000000", Available: "10.000000000000000000", Reserved: "0"},
				},
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(3))
			Expect(resp.HasMore).To(BeFalse())

			// Verify BTC account
			Expect(resp.Accounts[0].Reference).To(Equal("BTC"))
			Expect(resp.Accounts[0].CreatedAt).To(Equal(bitstampLaunchDate))
			Expect(*resp.Accounts[0].DefaultAsset).To(Equal("BTC/8"))

			// Verify USD account
			Expect(resp.Accounts[1].Reference).To(Equal("USD"))
			Expect(*resp.Accounts[1].DefaultAsset).To(Equal("USD/2"))

			// Verify ETH account
			Expect(resp.Accounts[2].Reference).To(Equal("ETH"))
			Expect(*resp.Accounts[2].DefaultAsset).To(Equal("ETH/18"))
		})

		It("should skip zero-balance currencies", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetAccountBalances(gomock.Any()).Return(
				[]client.AccountBalance{
					{Currency: "btc", Total: "1.00000000", Available: "1.00000000", Reserved: "0"},
					{Currency: "usd", Total: "0", Available: "0", Reserved: "0"},
				},
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(1))
			Expect(resp.Accounts[0].Reference).To(Equal("BTC"))
		})

		It("should skip unsupported currencies", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetAccountBalances(gomock.Any()).Return(
				[]client.AccountBalance{
					{Currency: "unknown_coin", Total: "100", Available: "100", Reserved: "0"},
					{Currency: "btc", Total: "1.00000000", Available: "1.00000000", Reserved: "0"},
				},
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(1))
			Expect(resp.Accounts[0].Reference).To(Equal("BTC"))
		})

		It("should handle empty response", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetAccountBalances(gomock.Any()).Return(
				[]client.AccountBalance{},
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
		})
	})
})
