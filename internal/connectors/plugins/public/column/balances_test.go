package column

import (
	"errors"
	"math/big"

	"github.com/formancehq/payments/internal/connectors/plugins/public/column/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Column Plugin Balances", func() {
	var (
		m   *client.MockHTTPClient
		plg models.Plugin
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		m = client.NewMockHTTPClient(ctrl)
		c := client.New("test", "aseplye", "https://test.com")
		c.SetHttpClient(m)
		plg = &Plugin{client: c}
	})

	Context("fetching next balances", func() {
		var (
			sampleBalance *client.Balance
		)

		BeforeEach(func() {
			sampleBalance = &client.Balance{
				AvailableAmount: "1000",
				HoldingAmount:   "1000",
				LockedAmount:    "1000",
				PendingAmount:   "1000",
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
				FromPayload: []byte(`{"reference": "test"}`),
			}
			m.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				500,
				errors.New("test error"),
			)
			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to get balance: test error : "))
			Expect(resp).To(Equal(models.FetchNextBalancesResponse{}))
		})

		It("should fetch all balances", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				PageSize:    60,
				FromPayload: []byte(`{"reference": "test", "defaultAsset": "USD"}`),
			}
			m.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.BalanceResponseWrapper[*client.Balance]{
				Balances: sampleBalance,
			})
			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(1))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).To(BeNil())
			Expect(resp.Balances[0].Amount).To(Equal(big.NewInt(1000)))
		})
	})
})
