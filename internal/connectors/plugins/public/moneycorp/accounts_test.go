package moneycorp

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/moneycorp/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Moneycorp Plugin Accounts", func() {
	Context("fetch next accounts", func() {
		var (
			plg *Plugin
			m   *client.MockClient

			pageSize       int
			sampleAccounts []*client.Account
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{client: m}
			pageSize = 15

			sampleAccounts = make([]*client.Account, 0)
			for i := 0; i < pageSize; i++ {
				sampleAccounts = append(sampleAccounts, &client.Account{
					ID: fmt.Sprintf("moneycorp-reference-%d", i),
				})
			}

		})
		It("fetches next accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    json.RawMessage(`{}`),
				PageSize: pageSize,
			}
			m.EXPECT().GetAccounts(ctx, gomock.Any(), pageSize).Return(
				sampleAccounts,
				nil,
			)
			res, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.HasMore).To(BeTrue())
			Expect(res.Accounts).To(HaveLen(req.PageSize))

			var state accountsState

			err = json.Unmarshal(res.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastPage).To(Equal(0))
			Expect(state.LastIDCreated).To(Equal(res.Accounts[len(res.Accounts)-1].Reference))
		})
	})
})
