package increase

import (
	"errors"
	"math/big"

	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Increase Plugin Balances", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("fetching next balances", func() {
		var (
			m             *client.MockClient
			sampleAccount *client.Account
			sampleBalance *client.Balance
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m

			sampleBalance = &client.Balance{
				AccountID:        "test_id",
				CurrentBalance:   "1000",
				AvailableBalance: "1000",
			}

			sampleAccount = &client.Account{
				Name:     "Test",
				Currency: "USD",
			}
		})

		It("should return an error - missing payload", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				PageSize: 60,
			}

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("missing from payload in request"))
			Expect(resp).To(Equal(models.FetchNextBalancesResponse{}))
		})

		It("should return an error - get balances error", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				PageSize:    60,
				FromPayload: []byte(`{"account_id": "test"}`),
			}

			m.EXPECT().GetAccountBalance(gomock.Any(), "test").Return(
				sampleBalance,
				errors.New("test error"),
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextBalancesResponse{}))
		})

		It("should fetch all balances", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				PageSize:    60,
				FromPayload: []byte(`{"account_id": "test"}`),
			}

			m.EXPECT().GetAccountBalance(gomock.Any(), "test").Return(
				sampleBalance,
				nil,
			)

			m.EXPECT().GetAccount(gomock.Any(), "test").Return(
				sampleAccount,
				nil,
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(1))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).To(BeNil())

			Expect(resp.Balances[0].Amount).To(Equal(big.NewInt(100000)))
		})
	})
})
