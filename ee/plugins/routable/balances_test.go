package routable

import (
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Routable fetchNextBalances", func() {
	var (
		ctrl   *gomock.Controller
		mock   *client.MockClient
		plg    *Plugin
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mock = client.NewMockClient(ctrl)
		plg = &Plugin{Plugin: plugins.NewBasePlugin(), name: "routable", logger: logger, client: mock}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("returns the available balance for the supplied account", func(ctx SpecContext) {
		mock.EXPECT().GetAccount(gomock.Any(), "acc_1").Return(&client.Account{
			ID:           "acc_1",
			CurrencyCode: "USD",
			TypeDetails:  client.AccountTypeDetails{AvailableAmount: "12.34"},
		}, nil)

		from := models.PSPAccount{Reference: "acc_1", CreatedAt: time.Now().UTC()}
		raw, _ := json.Marshal(from)
		resp, err := plg.fetchNextBalances(ctx, models.FetchNextBalancesRequest{FromPayload: raw})
		Expect(err).To(BeNil())
		Expect(resp.Balances).To(HaveLen(1))
		Expect(resp.Balances[0].AccountReference).To(Equal("acc_1"))
		Expect(resp.Balances[0].Amount.String()).To(Equal("1234"))
	})

	It("rejects requests with no from payload", func(ctx SpecContext) {
		_, err := plg.fetchNextBalances(ctx, models.FetchNextBalancesRequest{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing from payload"))
	})

	It("emits no balance when the currency is unsupported (logged + skipped)", func(ctx SpecContext) {
		mock.EXPECT().GetAccount(gomock.Any(), "acc_x").Return(&client.Account{
			ID:           "acc_x",
			CurrencyCode: "ZZZ",
			TypeDetails:  client.AccountTypeDetails{AvailableAmount: "5.00"},
		}, nil)
		from := models.PSPAccount{Reference: "acc_x", CreatedAt: time.Now().UTC()}
		raw, _ := json.Marshal(from)
		resp, err := plg.fetchNextBalances(ctx, models.FetchNextBalancesRequest{FromPayload: raw})
		Expect(err).To(BeNil())
		Expect(resp.Balances).To(BeEmpty())
	})
})
