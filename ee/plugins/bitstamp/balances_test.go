package bitstamp

import (
	"errors"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
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

	Context("fetching next balances", func() {
		It("returns balances for all supported currencies", func(ctx SpecContext) {
			m.EXPECT().GetAccountBalances(gomock.Any()).Return([]client.AccountBalance{
				{Currency: "btc", Available: "1.50000000"},
				{Currency: "usd", Available: "100.00"},
			}, nil)

			resp, err := plg.FetchNextBalances(ctx, models.FetchNextBalancesRequest{})
			Expect(err).To(BeNil())
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.Balances).To(HaveLen(2))

			refs := []string{resp.Balances[0].AccountReference, resp.Balances[1].AccountReference}
			Expect(refs).To(ConsistOf("BTC", "USD"))
		})

		It("skips unsupported currencies silently", func(ctx SpecContext) {
			m.EXPECT().GetAccountBalances(gomock.Any()).Return([]client.AccountBalance{
				{Currency: "xyz", Available: "1.0"},
				{Currency: "btc", Available: "0.5"},
			}, nil)

			resp, err := plg.FetchNextBalances(ctx, models.FetchNextBalancesRequest{})
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(1))
			Expect(resp.Balances[0].AccountReference).To(Equal("BTC"))
		})

		It("propagates GetAccountBalances errors", func(ctx SpecContext) {
			expectedErr := errors.New("failed")
			m.EXPECT().GetAccountBalances(gomock.Any()).Return(nil, expectedErr)

			_, err := plg.FetchNextBalances(ctx, models.FetchNextBalancesRequest{})
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expectedErr))
		})
	})
})
