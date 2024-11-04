package moneycorp

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/formancehq/payments/internal/connectors/plugins/public/moneycorp/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Moneycorp Plugin Balances", func() {
	var (
		plg *Plugin
	)

	Context("fetch next balances", func() {
		var (
			m *client.MockClient

			accRef         string
			sampleBalance  *client.Balance
			expectedAmount *big.Int
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{client: m}

			accRef = "abc"
			expectedAmount = big.NewInt(309900)
			sampleBalance = &client.Balance{
				Attributes: client.Attributes{
					CurrencyCode:     "AED",
					AvailableBalance: json.Number("3099"),
				},
			}
		})
		It("fetches next balances", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				FromPayload: json.RawMessage(fmt.Sprintf(`{"reference": "%s"}`, accRef)),
				State:       json.RawMessage(`{}`),
			}
			m.EXPECT().GetAccountBalances(gomock.Any(), accRef).Return(
				[]*client.Balance{sampleBalance},
				nil,
			)
			res, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Balances).To(HaveLen(1))

			Expect(res.Balances[0].AccountReference).To(Equal(accRef))
			Expect(res.Balances[0].Amount).To(BeEquivalentTo(expectedAmount))
			Expect(res.Balances[0].Asset).To(HavePrefix(strings.ToUpper(sampleBalance.Attributes.CurrencyCode)))
		})
	})
})
