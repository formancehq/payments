package stripe

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/formancehq/payments/internal/connectors/plugins/public/stripe/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	stripesdk "github.com/stripe/stripe-go/v79"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Stripe Plugin Balances", func() {
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

	Context("fetch next balances", func() {
		var (
			accRef        string
			sampleBalance *stripesdk.Balance
		)

		BeforeEach(func() {
			accRef = "abc"
			sampleBalance = &stripesdk.Balance{
				Available: []*stripesdk.Amount{
					{
						Currency: stripesdk.CurrencyAED,
						Amount:   49999,
					},
				},
			}
		})
		It("fetches next balances", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				FromPayload: json.RawMessage(fmt.Sprintf(`{"reference": "%s"}`, accRef)),
				State:       json.RawMessage(`{}`),
			}
			m.EXPECT().GetAccountBalances(gomock.Any(), accRef).Return(
				sampleBalance,
				nil,
			)
			res, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Balances).To(HaveLen(len(sampleBalance.Available)))

			for i, available := range sampleBalance.Available {
				Expect(res.Balances[i].AccountReference).To(Equal(accRef))
				Expect(res.Balances[i].Amount).To(BeEquivalentTo(big.NewInt(available.Amount)))
				Expect(res.Balances[i].Asset).To(HavePrefix(strings.ToUpper(string(available.Currency))))
			}
		})
	})
})
