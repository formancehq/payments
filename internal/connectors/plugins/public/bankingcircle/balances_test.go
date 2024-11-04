package bankingcircle

import (
	"errors"

	"github.com/formancehq/payments/internal/connectors/plugins/public/bankingcircle/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("BankingCircle Plugin Balances", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("fetching next balances", func() {
		var (
			m              *client.MockClient
			sampleBalances []client.Balance
			sampleAccount  *client.Account
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m

			sampleBalances = []client.Balance{
				{
					Currency:         "EUR",
					BeginOfDayAmount: "100",
					IntraDayAmount:   "150",
				},
				{
					Currency:         "USD",
					BeginOfDayAmount: "100",
					IntraDayAmount:   "-100",
				},
			}

			sampleAccount = &client.Account{
				Balances: sampleBalances,
			}
		})

		It("should return an error - get balances error", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				PageSize:    60,
				FromPayload: []byte(`{"reference": "123"}`),
			}

			m.EXPECT().GetAccount(gomock.Any(), "123").Return(
				sampleAccount,
				errors.New("test error"),
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextBalancesResponse{}))
		})

		It("should fetch next balances - no state no results", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				PageSize:    60,
				FromPayload: []byte(`{"reference": "123"}`),
			}

			m.EXPECT().GetAccount(gomock.Any(), "123").Return(
				&client.Account{},
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
				PageSize:    60,
				FromPayload: []byte(`{"reference": "123"}`),
			}

			m.EXPECT().GetAccount(gomock.Any(), "123").Return(
				sampleAccount,
				nil,
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(2))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())
		})

		It("should fetch all balances - page size < sample balances", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				PageSize:    1,
				FromPayload: []byte(`{"reference": "123"}`),
			}

			m.EXPECT().GetAccount(gomock.Any(), "123").Return(
				sampleAccount,
				nil,
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(2))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())
		})
	})
})
