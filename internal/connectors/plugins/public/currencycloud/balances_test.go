package currencycloud

import (
	"errors"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/currencycloud/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("CurrencyCloud Plugin Balances", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("fetching next balances", func() {
		var (
			m              *client.MockClient
			sampleBalances []*client.Balance
			now            time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
			now = time.Now().UTC()

			sampleBalances = []*client.Balance{
				{
					ID:        "test1",
					AccountID: "test1",
					Currency:  "EUR",
					Amount:    "100",
					CreatedAt: now.Add(-time.Duration(50) * time.Minute).UTC(),
					UpdatedAt: now.Add(-time.Duration(50) * time.Minute).UTC(),
				},
				{
					ID:        "test2",
					AccountID: "test2",
					Currency:  "USD",
					Amount:    "200",
					CreatedAt: now.Add(-time.Duration(40) * time.Minute).UTC(),
					UpdatedAt: now.Add(-time.Duration(35) * time.Minute).UTC(),
				},
				{
					ID:        "test3",
					AccountID: "test1",
					Currency:  "DKK",
					Amount:    "150",
					CreatedAt: now.Add(-time.Duration(30) * time.Minute).UTC(),
					UpdatedAt: now.Add(-time.Duration(15) * time.Minute).UTC(),
				},
			}
		})

		It("should return an error - get balances error", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				PageSize: 60,
			}

			m.EXPECT().GetBalances(ctx, 1, 60).Return(
				[]*client.Balance{},
				-1,
				errors.New("test error"),
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextBalancesResponse{}))
		})

		It("should fetch next balances - no state no results", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				PageSize: 60,
			}

			m.EXPECT().GetBalances(ctx, 1, 60).Return(
				[]*client.Balance{},
				-1,
				nil,
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())
		})

		It("should fetch all balances - page size > sample balances", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				PageSize: 60,
			}

			m.EXPECT().GetBalances(ctx, 1, 60).Return(
				sampleBalances,
				-1,
				nil,
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(3))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())
		})

		It("should fetch all balances - page size < sample balances", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				PageSize: 2,
			}

			m.EXPECT().GetBalances(ctx, 1, 2).Return(
				sampleBalances[:2],
				2,
				nil,
			)

			m.EXPECT().GetBalances(ctx, 2, 2).Return(
				sampleBalances[2:],
				-1,
				nil,
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(3))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())
		})
	})
})
