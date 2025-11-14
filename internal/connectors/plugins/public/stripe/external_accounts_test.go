package stripe

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/stripe/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	stripesdk "github.com/stripe/stripe-go/v79"
	gomock "github.com/golang/mock/gomock"
)

var _ = Describe("Stripe Plugin ExternalAccounts", func() {
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
		It("fetches next ExternalAccounts", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				FromPayload: json.RawMessage(fmt.Sprintf(`{"reference": "%s"}`, accRef)),
				State:       json.RawMessage(`{}`),
				PageSize:    pageSize,
			}
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
