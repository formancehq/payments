package moneycorp

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	logging "github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/ce/plugins/moneycorp/client"
	"github.com/formancehq/payments/pkg/domain/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Moneycorp *Plugin Balances", func() {
	var (
		plg models.Plugin
	)

	Context("fetch next balances", func() {
		var (
			ctrl *gomock.Controller
			m    *client.MockClient

			accRef         string
			sampleBalance  *client.Balance
			expectedAmount *big.Int
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{client: m, logger: logging.NewDefaultLogger(GinkgoWriter, true, false, false)}

			accRef = "abc"
			expectedAmount = big.NewInt(309900)
			sampleBalance = &client.Balance{
				Attributes: client.Attributes{
					CurrencyCode:     "AED",
					AvailableBalance: json.Number("3099"),
				},
			}
		})

		AfterEach(func() {
			ctrl.Finish()
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

		It("skips balances with unsupported currencies", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				FromPayload: json.RawMessage(fmt.Sprintf(`{"reference": "%s"}`, accRef)),
				State:       json.RawMessage(`{}`),
			}
			unsupported := &client.Balance{
				Attributes: client.Attributes{
					CurrencyCode:     "ZZZ",
					AvailableBalance: json.Number("100"),
				},
			}
			m.EXPECT().GetAccountBalances(gomock.Any(), accRef).Return(
				[]*client.Balance{unsupported, sampleBalance},
				nil,
			)
			res, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			// Only the supported balance is returned; the unsupported one is skipped.
			Expect(res.Balances).To(HaveLen(1))
			Expect(res.Balances[0].Asset).To(HavePrefix(strings.ToUpper(sampleBalance.Attributes.CurrencyCode)))
		})
	})
})
