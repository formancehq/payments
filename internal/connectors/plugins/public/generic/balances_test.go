package generic

import (
	"errors"
	"math/big"
	"time"

	"github.com/formancehq/payments/genericclient"
	"github.com/formancehq/payments/internal/connectors/plugins/public/generic/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/golang/mock/gomock"
)

var _ = Describe("Generic Plugin Balances", func() {
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
			sampleBalance genericclient.Balances
			now           time.Time
		)

		BeforeEach(func() {
			now = time.Now().UTC()

			sampleBalance = genericclient.Balances{
				Id:        "1",
				AccountID: "123",
				At:        now,
				Balances: []genericclient.Balance{
					{
						Amount:   "100",
						Currency: "USD",
					},
					{
						Amount:   "15001",
						Currency: "EUR",
					},
				},
			}
		})

		It("should return an error - missing payload", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				PageSize: 60,
			}

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("from payload is required: invalid request"))
			Expect(resp).To(Equal(models.FetchNextBalancesResponse{}))
		})

		It("should return an error - get balances error", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				PageSize:    60,
				FromPayload: []byte(`{"reference": "test"}`),
			}

			m.EXPECT().GetBalances(gomock.Any(), "test").Return(
				&sampleBalance,
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
				FromPayload: []byte(`{"reference": "test"}`),
			}

			m.EXPECT().GetBalances(gomock.Any(), "test").Return(
				&sampleBalance,
				nil,
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(2))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).To(BeNil())
			Expect(resp.Balances[0].Amount).To(Equal(big.NewInt(100)))
			Expect(resp.Balances[0].Asset).To(Equal("USD/2"))
			Expect(resp.Balances[1].Amount).To(Equal(big.NewInt(15001)))
			Expect(resp.Balances[1].Asset).To(Equal("EUR/2"))
		})
	})
})
