package wise

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/wise/client"
	"github.com/formancehq/payments/internal/models"
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Wise Plugin External Accounts", func() {
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

	Context("fetching next external accounts", func() {
		var (
			sampleRecipientAccounts []*client.RecipientAccount
		)

		BeforeEach(func() {
			sampleRecipientAccounts = make([]*client.RecipientAccount, 0)
			for i := 0; i < 50; i++ {
				sampleRecipientAccounts = append(sampleRecipientAccounts, &client.RecipientAccount{
					ID:       uint64(i),
					Profile:  uint64(0),
					Currency: "USD",
					Name: client.Name{
						FullName: fmt.Sprintf("Account %d", i),
					},
				})
			}
		})

		It("should return an error - get beneficiaries error", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				PageSize:    60,
				FromPayload: []byte(`{"id": 0}`),
			}

			m.EXPECT().GetRecipientAccounts(gomock.Any(), uint64(0), 60, uint64(0)).Return(
				&client.RecipientAccountsResponse{},
				errors.New("test error"),
			)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextExternalAccountsResponse{}))
		})

		It("should fetch next external accounts - no state no results", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				PageSize:    60,
				FromPayload: []byte(`{"id": 0}`),
			}

			m.EXPECT().GetRecipientAccounts(gomock.Any(), uint64(0), 60, uint64(0)).Return(
				&client.RecipientAccountsResponse{
					Content:             []*client.RecipientAccount{},
					SeekPositionForNext: 0,
				},
				nil,
			)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state externalAccountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.LastSeekPosition).To(Equal(uint64(0)))
		})

		It("should fetch next external accounts - no state pageSize > total accounts", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				PageSize:    60,
				FromPayload: []byte(`{"id": 0}`),
			}

			m.EXPECT().GetRecipientAccounts(gomock.Any(), uint64(0), 60, uint64(0)).Return(
				&client.RecipientAccountsResponse{
					Content:             sampleRecipientAccounts,
					SeekPositionForNext: 0,
				},
				nil,
			)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(50))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state externalAccountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastSeekPosition).To(Equal(uint64(49)))
		})

		It("should fetch next accounts - no state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				PageSize:    40,
				FromPayload: []byte(`{"id": 0}`),
			}

			m.EXPECT().GetRecipientAccounts(gomock.Any(), uint64(0), 40, uint64(0)).Return(
				&client.RecipientAccountsResponse{
					Content:             sampleRecipientAccounts[:40],
					SeekPositionForNext: 39,
				},
				nil,
			)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(40))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())

			var state externalAccountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastSeekPosition).To(Equal(uint64(39)))
		})

		It("should fetch next external accounts - with state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:       []byte(fmt.Sprintf(`{"lastSeekPosition": %d}`, 38)),
				PageSize:    40,
				FromPayload: []byte(`{"id": 0}`),
			}

			m.EXPECT().GetRecipientAccounts(gomock.Any(), uint64(0), 40, uint64(38)).Return(
				&client.RecipientAccountsResponse{
					Content:             sampleRecipientAccounts[39:],
					SeekPositionForNext: 49,
				},
				nil,
			)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(11))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state externalAccountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.LastSeekPosition).To(Equal(uint64(49)))
		})
	})
})
