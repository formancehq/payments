package krakenpro

import (
	"encoding/json"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Krakenpro Accounts", func() {
	var (
		p      *Plugin
		m      *client.MockClient
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		p = &Plugin{
			Plugin: plugins.NewBasePlugin(),
			client: m,
			logger: logger,
			config: Config{
				APIKey: "test-api-key",
			},
			accountRef: "kraken-test12345",
			currencies: map[string]int{"USD": 2, "BTC": 8},
		}
	})

	Context("fetch next accounts", func() {
		It("should return a single account on first call", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    json.RawMessage(`{}`),
				PageSize: 25,
			}

			resp, err := p.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(1))
			Expect(resp.Accounts[0].Reference).To(Equal("kraken-test12345"))
			Expect(*resp.Accounts[0].Name).To(Equal("Kraken Pro"))
			Expect(resp.Accounts[0].Metadata["provider"]).To(Equal("krakenpro"))
			Expect(resp.HasMore).To(BeFalse())
		})

		It("should return no accounts on subsequent calls", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    json.RawMessage(`{"fetched": true}`),
				PageSize: 25,
			}

			resp, err := p.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(BeEmpty())
			Expect(resp.HasMore).To(BeFalse())
		})
	})
})
