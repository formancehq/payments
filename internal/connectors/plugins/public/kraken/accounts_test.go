package kraken

import (
	"encoding/json"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/public/kraken/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Kraken Plugin Accounts", func() {
	var (
		ctrl   *gomock.Controller
		m      *client.MockClient
		plg    *Plugin
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{
			client: m,
			logger: logger,
			config: Config{
				Endpoint: "https://api.kraken.com",
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetch next accounts", func() {
		It("returns a single main account", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    json.RawMessage(`{}`),
				PageSize: 20,
			}

			res, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.HasMore).To(BeFalse())
			Expect(res.Accounts).To(HaveLen(1))
			Expect(res.Accounts[0].Reference).To(Equal("main"))
			Expect(*res.Accounts[0].Name).To(Equal("Kraken Main Account"))
			Expect(res.Accounts[0].Metadata["provider"]).To(Equal("kraken"))
		})
	})
})
