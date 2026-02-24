package stripe

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/pkg/connectors/stripe/client"
	"github.com/formancehq/payments/pkg/connector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	stripesdk "github.com/stripe/stripe-go/v80"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Stripe Plugin ExternalAccounts", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  connector.Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m, logger: logging.Testing()}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetch next ExternalAccounts", func() {
		var (
			pageSize               int
			sampleExternalAccounts []*stripesdk.BankAccount
			accRef                 string
			created                int64
		)

		BeforeEach(func() {
			pageSize = 10
			accRef = "baseAcc"
			created = 1483565364
			sampleExternalAccounts = make([]*stripesdk.BankAccount, 0)
			for i := 0; i < pageSize; i++ {
				if i%2 == 0 {
					created = 0
				}
				sampleExternalAccounts = append(sampleExternalAccounts, &stripesdk.BankAccount{
					ID:      fmt.Sprintf("some-reference-%d", i),
					Account: &stripesdk.Account{Created: created},
				})
			}

		})

		It("skips fetching ExternalAccounts when from is the root account", func(ctx SpecContext) {
			rootAccountID := "someRoot"
			req := connector.FetchNextExternalAccountsRequest{
				FromPayload: json.RawMessage(fmt.Sprintf(`{"reference": "%s"}`, rootAccountID)),
				State:       json.RawMessage(`{}`),
				PageSize:    pageSize,
			}
			m.EXPECT().GetRootAccountID().Return(rootAccountID)
			res, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.HasMore).To(BeFalse())
		})

		It("fetches next ExternalAccounts", func(ctx SpecContext) {
			req := connector.FetchNextExternalAccountsRequest{
				FromPayload: json.RawMessage(fmt.Sprintf(`{"reference": "%s"}`, accRef)),
				State:       json.RawMessage(`{}`),
				PageSize:    pageSize,
			}
			m.EXPECT().GetRootAccountID().Return("roooooot")
			m.EXPECT().GetExternalAccounts(gomock.Any(), accRef, gomock.Any(), int64(pageSize)).Return(
				sampleExternalAccounts,
				client.Timeline{LatestID: sampleExternalAccounts[len(sampleExternalAccounts)-1].ID},
				true,
				nil,
			)
			res, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.HasMore).To(BeTrue())
			Expect(res.ExternalAccounts).To(HaveLen(pageSize))

			for _, acc := range res.ExternalAccounts {
				Expect(acc.CreatedAt.IsZero()).To(BeFalse())
				Expect(acc.CreatedAt).NotTo(Equal(time.Unix(0, 0).UTC()))
			}

			var state accountsState

			err = json.Unmarshal(res.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.Timeline.LatestID).To(Equal(res.ExternalAccounts[len(res.ExternalAccounts)-1].Reference))
		})
	})
})
