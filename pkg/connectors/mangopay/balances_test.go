package mangopay

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/formancehq/payments/pkg/connectors/mangopay/client"
	"github.com/formancehq/payments/pkg/connector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Mangopay Plugin Balances", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  connector.Plugin
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
			sampleBalance client.Wallet
		)

		BeforeEach(func() {

			sampleBalance = client.Wallet{
				Balance: struct {
					Currency string      `json:"Currency"`
					Amount   json.Number `json:"Amount"`
				}{
					Currency: "EUR",
					Amount:   "100",
				},
			}
		})

		It("should return an error - missing payload", func(ctx SpecContext) {
			req := connector.FetchNextBalancesRequest{
				PageSize: 60,
			}

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("missing from payload in request"))
			Expect(resp).To(Equal(connector.FetchNextBalancesResponse{}))
		})

		It("should return an error - get wallet error", func(ctx SpecContext) {
			req := connector.FetchNextBalancesRequest{
				PageSize:    60,
				FromPayload: []byte(`{"reference": "test"}`),
			}

			m.EXPECT().GetWallet(gomock.Any(), "test").Return(
				&sampleBalance,
				errors.New("test error"),
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(connector.FetchNextBalancesResponse{}))
		})

		It("should fetch all balances", func(ctx SpecContext) {
			req := connector.FetchNextBalancesRequest{
				PageSize:    60,
				FromPayload: []byte(`{"reference": "test"}`),
			}

			m.EXPECT().GetWallet(gomock.Any(), "test").Return(
				&sampleBalance,
				nil,
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(1))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).To(BeNil())
			Expect(resp.Balances[0].Amount).To(Equal(big.NewInt(100)))
			Expect(resp.Balances[0].Asset).To(Equal("EUR/2"))
			Expect(resp.Balances[0].AccountReference).To(Equal("test"))
		})
	})
})
