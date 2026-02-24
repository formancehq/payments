package modulr

import (
	"errors"
	"math/big"

	"github.com/formancehq/payments/pkg/connectors/modulr/client"
	"github.com/formancehq/payments/pkg/connector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Modulr Plugin Balances", func() {
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
			sampleBalance client.Account
		)

		BeforeEach(func() {
			sampleBalance = client.Account{
				Balance:  "150.01",
				Currency: "USD",
			}
		})

		It("should return an error - missing payload", func(ctx SpecContext) {
			req := connector.FetchNextBalancesRequest{
				PageSize: 60,
			}

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("missing from payload when fetching balances"))
			Expect(resp).To(Equal(connector.FetchNextBalancesResponse{}))
		})

		It("should return an error - get balances error", func(ctx SpecContext) {
			req := connector.FetchNextBalancesRequest{
				PageSize:    60,
				FromPayload: []byte(`{"reference": "test"}`),
			}

			m.EXPECT().GetAccount(gomock.Any(), "test").Return(
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

			m.EXPECT().GetAccount(gomock.Any(), "test").Return(
				&sampleBalance,
				nil,
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(1))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).To(BeNil())
			Expect(resp.Balances[0].Amount).To(Equal(big.NewInt(15001)))
		})
	})
})
