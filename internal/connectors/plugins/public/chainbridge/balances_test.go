package chainbridge

import (
	"errors"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/chainbridge/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("ChainBridge Plugin Balances", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  models.Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetching next balances", func() {
		var (
			sampleBalances []*client.TokenBalance
		)

		BeforeEach(func() {
			now := time.Now().UTC()

			sampleBalances = []*client.TokenBalance{
				{
					MonitorID: "mon_1",
					Asset:     "ETH/18",
					Amount:    big.NewInt(1000000000000000000),
					FetchedAt: now,
				},
				{
					MonitorID: "mon_1",
					Asset:     "USDC/6",
					Amount:    big.NewInt(5000000),
					FetchedAt: now,
				},
				{
					MonitorID: "mon_2",
					Asset:     "invalid asset!",
					Amount:    big.NewInt(100),
					FetchedAt: now,
				},
				{
					MonitorID: "mon_2",
					Asset:     "BTC/8",
					Amount:    big.NewInt(50000000),
					FetchedAt: now,
				},
			}
		})

		It("should return an error - get balances error", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				PageSize: 60,
			}

			m.EXPECT().GetBalances(gomock.Any()).Return(
				nil,
				errors.New("test error"),
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(resp).To(Equal(models.FetchNextBalancesResponse{}))
		})

		It("should fetch next balances - no results", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				PageSize: 60,
			}

			m.EXPECT().GetBalances(gomock.Any()).Return(
				[]*client.TokenBalance{},
				nil,
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
		})

		It("should fetch balances and skip invalid assets", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				PageSize: 60,
			}

			m.EXPECT().GetBalances(gomock.Any()).Return(
				sampleBalances,
				nil,
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(3))
			Expect(resp.HasMore).To(BeFalse())

			// Verify that "invalid asset!" was skipped
			for _, b := range resp.Balances {
				Expect(b.Asset).ToNot(Equal("invalid asset!"))
			}

			// Verify balance mapping
			Expect(resp.Balances[0].AccountReference).To(Equal("mon_1"))
			Expect(resp.Balances[0].Asset).To(Equal("ETH/18"))
			Expect(resp.Balances[0].Amount).To(Equal(big.NewInt(1000000000000000000)))

			Expect(resp.Balances[1].AccountReference).To(Equal("mon_1"))
			Expect(resp.Balances[1].Asset).To(Equal("USDC/6"))

			Expect(resp.Balances[2].AccountReference).To(Equal("mon_2"))
			Expect(resp.Balances[2].Asset).To(Equal("BTC/8"))
		})
	})
})
