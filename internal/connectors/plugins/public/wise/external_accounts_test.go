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

var _ = Describe("Wise Plugin External Accounts", func() {
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

	Context("fetch next external accounts", func() {
		var (
			accounts          []*client.RecipientAccount
			expectedProfileID uint64
			lastSeekPosition  uint64
		)

		BeforeEach(func() {
			expectedProfileID = 154
			lastSeekPosition = 83
			accounts = []*client.RecipientAccount{
				{ID: lastSeekPosition + 1, Profile: expectedProfileID},
				{ID: lastSeekPosition + 2, Profile: expectedProfileID},
			}
		})

		It("fetches recpient accounts from wise", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State: json.RawMessage(fmt.Sprintf(`{"lastSeekPosition":%d}`, lastSeekPosition)),
				FromPayload: json.RawMessage(fmt.Sprintf(
					`{"id":%d,"type":"sometype"}`,
					expectedProfileID,
				)),
				PageSize: len(accounts),
			}
			recipientRes := &client.RecipientAccountsResponse{
				Content:                accounts,
				SeekPositionForCurrent: lastSeekPosition + 1,
				SeekPositionForNext:    accounts[len(accounts)-1].ID + 1,
			}
			m.EXPECT().GetRecipientAccounts(ctx, expectedProfileID, req.PageSize, lastSeekPosition).Return(recipientRes, nil)

			res, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.HasMore).To(BeTrue())
			Expect(res.ExternalAccounts).To(HaveLen(req.PageSize))
			Expect(res.ExternalAccounts[0].Reference).To(Equal(fmt.Sprint(accounts[0].ID)))

			var state externalAccountsState

			err = json.Unmarshal(res.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastSeekPosition).To(Equal(recipientRes.SeekPositionForNext))
		})
	})
})
