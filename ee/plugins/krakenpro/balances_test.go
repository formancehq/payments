package krakenpro

import (
	"errors"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Kraken Pro fetch_balances", func() {
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
				"BTC": 8, "USD": 2,
			},
			assetsLoaded: time.Now(),
		}
	})

	AfterEach(func() { ctrl.Finish() })

	It("emits one PSPBalance per raw variant with the real per-variant amount", func(ctx SpecContext) {
		// XXBT (spot) and XBT.M (earn) are NOT summed — each reports its
		// own balance against its own per-class account reference.
		m.EXPECT().GetBalanceEx(gomock.Any()).Return(map[string]client.BalanceExEntry{
			"XXBT":  {Balance: "2.0", HoldTrade: "0.5"},
			"XBT.M": {Balance: "0.3", HoldTrade: "0"},
			"ZUSD":  {Balance: "100.00", HoldTrade: "25.00"},
		}, nil)

		resp, err := plg.FetchNextBalances(ctx, models.FetchNextBalancesRequest{})
		Expect(err).To(BeNil())
		Expect(resp.Balances).To(HaveLen(3))

		byRef := map[string]*models.PSPBalance{}
		for i := range resp.Balances {
			byRef[resp.Balances[i].AccountReference] = &resp.Balances[i]
		}
		// Reference is the raw code; Asset is the normalised symbol.
		Expect(byRef["XXBT"].Amount.Cmp(big.NewInt(150_000_000))).To(Equal(0)) // 1.5 BTC spot
		Expect(byRef["XXBT"].Asset).To(Equal("BTC/8"))
		Expect(byRef["XBT.M"].Amount.Cmp(big.NewInt(30_000_000))).To(Equal(0)) // 0.3 BTC earn
		Expect(byRef["XBT.M"].Asset).To(Equal("BTC/8"))
		Expect(byRef["ZUSD"].Amount.Cmp(big.NewInt(7500))).To(Equal(0)) // 75.00 USD
	})

	It("skips unknown assets", func(ctx SpecContext) {
		m.EXPECT().GetBalanceEx(gomock.Any()).Return(map[string]client.BalanceExEntry{
			"XYZ":  {Balance: "1.0"},
			"XXBT": {Balance: "0.5"},
		}, nil)
		resp, err := plg.FetchNextBalances(ctx, models.FetchNextBalancesRequest{})
		Expect(err).To(BeNil())
		Expect(resp.Balances).To(HaveLen(1))
		Expect(resp.Balances[0].AccountReference).To(Equal("XXBT"))
	})

	It("propagates BalanceEx errors", func(ctx SpecContext) {
		m.EXPECT().GetBalanceEx(gomock.Any()).Return(nil, errors.New("nope"))
		_, err := plg.FetchNextBalances(ctx, models.FetchNextBalancesRequest{})
		Expect(err).To(HaveOccurred())
	})
})
