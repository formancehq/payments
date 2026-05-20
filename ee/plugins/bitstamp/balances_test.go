package bitstamp

import (
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Bitstamp Plugin Balances", func() {
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
				"BTC": 8,
				"ETH": 18,
			},
			currLastSync: time.Now(),
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	// Balances is FromPayload-driven (Qonto pattern): the engine passes
	// the parent PSPAccount on req.FromPayload and the task derives
	// the balance from PSPAccount.Raw — no extra GetAccountBalances call.

	Context("fetching next balances", func() {
		mkParent := func(bal client.AccountBalance) []byte {
			raw, err := json.Marshal(bal)
			Expect(err).ToNot(HaveOccurred())
			asset := "BTC/8"
			parent := models.PSPAccount{
				Reference:    "BTC",
				DefaultAsset: &asset,
				Raw:          raw,
			}
			payload, err := json.Marshal(parent)
			Expect(err).ToNot(HaveOccurred())
			return payload
		}

		It("derives the balance from FromPayload without calling the API", func(ctx SpecContext) {
			payload := mkParent(client.AccountBalance{
				Currency:  "btc",
				Total:     "1.50000000",
				Available: "1.00000000",
				Reserved:  "0.50000000",
			})
			// NO m.EXPECT() call — the assertion is precisely that
			// no Bitstamp HTTP call is made during the balances task.

			resp, err := plg.FetchNextBalances(ctx, models.FetchNextBalancesRequest{FromPayload: payload})
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(1))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.Balances[0].AccountReference).To(Equal("BTC"))
			Expect(resp.Balances[0].Asset).To(Equal("BTC/8"))
			Expect(resp.Balances[0].Amount.Int64()).To(Equal(int64(100000000)))
		})

		It("errors when FromPayload is missing", func(ctx SpecContext) {
			resp, err := plg.FetchNextBalances(ctx, models.FetchNextBalancesRequest{})
			Expect(err).To(MatchError(models.ErrMissingFromPayloadInRequest))
			Expect(resp).To(Equal(models.FetchNextBalancesResponse{}))
		})

		It("errors on invalid FromPayload JSON", func(ctx SpecContext) {
			resp, err := plg.FetchNextBalances(ctx, models.FetchNextBalancesRequest{
				FromPayload: []byte(`{not-json`),
			})
			Expect(err).To(HaveOccurred())
			Expect(resp).To(Equal(models.FetchNextBalancesResponse{}))
		})

		It("skips unsupported currencies silently rather than erroring", func(ctx SpecContext) {
			payload := mkParent(client.AccountBalance{Currency: "xyz", Available: "1.0"})
			resp, err := plg.FetchNextBalances(ctx, models.FetchNextBalancesRequest{FromPayload: payload})
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(BeEmpty())
			Expect(resp.HasMore).To(BeFalse())
		})
	})
})
