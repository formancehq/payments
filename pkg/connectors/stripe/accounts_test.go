package stripe

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/pkg/connectors/stripe/client"
	"github.com/formancehq/payments/pkg/connector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	stripesdk "github.com/stripe/stripe-go/v80"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Stripe Plugin Accounts", func() {
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

	Context("fetch next accounts", func() {
		var (
			pageSize       int
			sampleAccounts []*stripesdk.Account
		)

		BeforeEach(func() {
			pageSize = 20

			sampleAccounts = make([]*stripesdk.Account, 0)
			for i := 0; i < pageSize-1; i++ {
				sampleAccounts = append(sampleAccounts, &stripesdk.Account{
					ID: fmt.Sprintf("some-reference-%d", i),
				})
			}

		})
		It("fetches next accounts", func(ctx SpecContext) {
			req := connector.FetchNextAccountsRequest{
				State:    json.RawMessage(`{}`),
				PageSize: pageSize,
			}
			rootAccount := &stripesdk.Account{
				ID: "root-account-id",
				Settings: &stripesdk.AccountSettings{
					Dashboard: &stripesdk.AccountSettingsDashboard{DisplayName: "displayName"}},
			}
			m.EXPECT().GetRootAccount().Return(
				rootAccount,
				nil,
			)
			// pageSize passed to client is less when we generate a root account
			m.EXPECT().GetAccounts(gomock.Any(), gomock.Any(), int64(pageSize-1)).Return(
				sampleAccounts,
				client.Timeline{LatestID: sampleAccounts[len(sampleAccounts)-1].ID},
				true,
				nil,
			)
			res, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.HasMore).To(BeTrue())
			Expect(res.Accounts).To(HaveLen(req.PageSize))
			Expect(res.Accounts[0].Reference).To(Equal(rootAccount.ID))
			Expect(res.Accounts[0].Name).To(Equal(&rootAccount.Settings.Dashboard.DisplayName))

			var state accountsState

			err = json.Unmarshal(res.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.Timeline.LatestID).To(Equal(res.Accounts[len(res.Accounts)-1].Reference))
		})
	})
})
