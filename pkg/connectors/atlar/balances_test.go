package atlar

import (
	"errors"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/pkg/connectors/atlar/client"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/get-momo/atlar-v1-go-client/client/accounts"
	atlar_models "github.com/get-momo/atlar-v1-go-client/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Atlar Plugin Balances", func() {
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
			sampleAccount atlar_models.Account
			now           time.Time
		)

		BeforeEach(func() {
			now = time.Now().UTC()

			sampleAccount = atlar_models.Account{
				Balance: &atlar_models.Balance{
					AccountID: "",
					Amount: &atlar_models.Amount{
						Currency:    pointer.For("EUR"),
						StringValue: pointer.For("100"),
						Value:       pointer.For(int64(100)),
					},
					Timestamp: now.Format(time.RFC3339Nano),
				},
			}
			_ = sampleAccount
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

		It("should return an error - get account error", func(ctx SpecContext) {
			req := connector.FetchNextBalancesRequest{
				PageSize:    60,
				FromPayload: []byte(`{"reference": "test"}`),
			}

			m.EXPECT().GetV1AccountsID(gomock.Any(), "test").Return(
				nil,
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

			m.EXPECT().GetV1AccountsID(gomock.Any(), "test").Return(
				&accounts.GetV1AccountsIDOK{
					Payload: &sampleAccount,
				},
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
