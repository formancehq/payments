package wise

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/wise/client"
	"github.com/formancehq/payments/internal/models"
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Wise Plugin Accounts", func() {
	var (
		plg *Plugin
		m   *client.MockClient
	)

	BeforeEach(func() {
		plg = &Plugin{}

		ctrl := gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg.SetClient(m)
	})

	Context("fetch next accounts", func() {
		var (
			balances          []client.Balance
			expectedProfileID uint64
		)

		BeforeEach(func() {
			expectedProfileID = 123454
			balances = []client.Balance{
				{ID: 14556, Type: "type1"},
				{ID: 3334, Type: "type2"},
			}
		})

		It("fetches accounts from wise", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:       json.RawMessage(`{}`),
				FromPayload: json.RawMessage(fmt.Sprintf(`{"ID":%d}`, expectedProfileID)),
				PageSize:    len(balances),
			}
			m.EXPECT().GetBalances(ctx, expectedProfileID).Return(
				balances,
				nil,
			)

			res, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.HasMore).To(BeTrue())
			Expect(res.Accounts).To(HaveLen(req.PageSize))
			Expect(res.Accounts[0].Reference).To(Equal(fmt.Sprint(balances[0].ID)))
			Expect(res.Accounts[1].Reference).To(Equal(fmt.Sprint(balances[1].ID)))

			var state accountsState

			err = json.Unmarshal(res.NewState, &state)
			Expect(err).To(BeNil())
			Expect(fmt.Sprint(state.LastAccountID)).To(Equal(res.Accounts[len(res.Accounts)-1].Reference))
		})
	})
})
