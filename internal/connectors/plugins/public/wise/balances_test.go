package wise

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/formancehq/payments/internal/connectors/plugins/public/wise/client"
	"github.com/formancehq/payments/internal/models"
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Wise Plugin Balances", func() {
	var (
		plg models.Plugin
		m   *client.MockClient
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m}
	})

	Context("fetch next balances", func() {
		var (
			balance           client.Balance
			expectedProfileID uint64
			profileVal        uint64
		)

		BeforeEach(func() {
			expectedProfileID = 123454
			profileVal = 999999
			balance = client.Balance{
				ID:     14556,
				Type:   "type1",
				Amount: client.BalanceAmount{Value: json.Number("44.99"), Currency: "USD"},
			}
		})

		It("fetches balances from wise client", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				State: json.RawMessage(`{}`),
				FromPayload: json.RawMessage(fmt.Sprintf(
					`{"Reference":"%d","Metadata":{"%s":"%d"}}`,
					expectedProfileID,
					metadataProfileIDKey,
					profileVal,
				)),
				PageSize: 10,
			}
			m.EXPECT().GetBalance(gomock.Any(), profileVal, expectedProfileID).Return(
				&balance,
				nil,
			)

			res, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.HasMore).To(BeFalse())
			Expect(res.Balances).To(HaveLen(1)) // always returns 1
			Expect(res.Balances[0].AccountReference).To(Equal(fmt.Sprint(expectedProfileID)))
			expectedBalance, err := balance.Amount.Value.Float64()
			Expect(err).To(BeNil())
			Expect(res.Balances[0].Amount).To(BeEquivalentTo(big.NewInt(int64(expectedBalance * 100))))
		})
	})
})
