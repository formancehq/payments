package moov

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/moov/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/moovfinancial/moov-go/pkg/moov"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Moov Users", func() {
	var (
		plg         *Plugin
		sampleUsers []moov.Account
	)

	BeforeEach(func() {
		plg = &Plugin{}
		sampleUsers = make([]moov.Account, 0)

		for i := 0; i < 50; i++ {
			sampleUsers = append(sampleUsers, moov.Account{
				AccountID:   fmt.Sprintf("%d", i),
				DisplayName: fmt.Sprintf("User %d", i),
				CreatedOn:   time.Now().UTC(),
			})
		}
	})

	Context("fetching next users", func() {
		var (
			m *client.MockClient
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
		})

		It("should return an error - invalid state payload", func(ctx SpecContext) {
			invalidPayload := []byte(`invalid json`)
			req := models.FetchNextOthersRequest{
				State: invalidPayload,
			}

			users, err := plg.fetchNextUsers(ctx, req)
			Expect(err).NotTo(BeNil())
			Expect(users.Others).To(HaveLen(0))
		})

		It("should return an error - get users error", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{
				Name:     fetchOthers,
				PageSize: 10,
			}

			m.EXPECT().GetUsers(gomock.Any(), 0, 10).Return(
				[]moov.Account{},
				errors.New("test error"),
			)

			resp, err := plg.fetchNextUsers(ctx, req)
			Expect(err).To(MatchError("test error"))
			Expect(resp.Others).To(HaveLen(0))
		})

		It("should fetch users with no results", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{
				Name:     fetchOthers,
				PageSize: 10,
			}

			m.EXPECT().GetUsers(gomock.Any(), 0, 10).Return(
				[]moov.Account{},
				nil,
			)

			resp, err := plg.fetchNextUsers(ctx, req)

			Expect(err).To(BeNil())
			Expect(resp.Others).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state usersState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.Skip).To(Equal(int64(0)))
		})

		It("should fetch users with state and return hasMore=true when pageSize equals results", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{
				Name:     fetchOthers,
				PageSize: 10,
				State:    json.RawMessage(`{"skip": 0}`),
			}

			m.EXPECT().GetUsers(gomock.Any(), 0, 10).Return(
				sampleUsers[:10],
				nil,
			)

			resp, err := plg.fetchNextUsers(ctx, req)

			Expect(err).To(BeNil())
			Expect(resp.Others).To(HaveLen(10))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())

			var state usersState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.Skip).To(Equal(int64(10)))

			Expect(resp.Others[0].ID).To(Equal("0"))

			var account moov.Account
			err = json.Unmarshal(resp.Others[0].Other, &account)
			Expect(err).To(BeNil())
			Expect(account.AccountID).To(Equal("0"))
		})

		It("should fetch users with pagination", func(ctx SpecContext) {
			req1 := models.FetchNextOthersRequest{
				Name:     fetchOthers,
				PageSize: 20,
				State:    json.RawMessage(`{"skip": 0}`),
			}

			m.EXPECT().GetUsers(gomock.Any(), 0, 20).Return(
				sampleUsers[:20],
				nil,
			)

			resp1, err := plg.fetchNextUsers(ctx, req1)

			Expect(err).To(BeNil())
			Expect(resp1.Others).To(HaveLen(20))
			Expect(resp1.HasMore).To(BeTrue())
			Expect(resp1.NewState).ToNot(BeNil())

			req2 := models.FetchNextOthersRequest{
				Name:     fetchOthers,
				PageSize: 20,
				State:    resp1.NewState,
			}

			m.EXPECT().GetUsers(gomock.Any(), 20, 20).Return(
				sampleUsers[20:40],
				nil,
			)

			resp2, err := plg.fetchNextUsers(ctx, req2)

			Expect(err).To(BeNil())
			Expect(resp2.Others).To(HaveLen(20))
			Expect(resp2.HasMore).To(BeTrue())
			Expect(resp2.NewState).ToNot(BeNil())

			req3 := models.FetchNextOthersRequest{
				Name:     fetchOthers,
				PageSize: 20,
				State:    resp2.NewState,
			}

			m.EXPECT().GetUsers(gomock.Any(), 40, 20).Return(
				sampleUsers[40:],
				nil,
			)

			resp3, err := plg.fetchNextUsers(ctx, req3)

			Expect(err).To(BeNil())
			Expect(resp3.Others).To(HaveLen(10))
			Expect(resp3.HasMore).To(BeFalse())
			Expect(resp3.NewState).ToNot(BeNil())
		})

		Context("fetch users with moov client", func() {
			var (
				mockedService *client.MockMoovClient
			)

			BeforeEach((func() {
				ctrl := gomock.NewController(GinkgoT())
				mockedService = client.NewMockMoovClient(ctrl)

				plg.client, _ = client.New("moov", "https://example.com", "access_token", "test", "test")
				plg.client.NewWithClient(mockedService)
			}))

			It("should fail when moov client returns an error", func(ctx SpecContext) {
				req := models.FetchNextOthersRequest{
					Name:     fetchOthers,
					PageSize: 10,
				}

				mockedService.EXPECT().GetMoovAccounts(gomock.Any(), 0, 10).Return(
					[]moov.Account{},
					errors.New("fetch users error"),
				)

				resp, err := plg.fetchNextUsers(ctx, req)

				Expect(err).To(MatchError("failed to get moov accounts: fetch users error"))
				Expect(resp.Others).To(HaveLen(0))
			})

			It("should handle empty result", func(ctx SpecContext) {
				req := models.FetchNextOthersRequest{
					Name:     fetchOthers,
					PageSize: 10,
				}

				mockedService.EXPECT().GetMoovAccounts(gomock.Any(), 0, 10).Return(
					[]moov.Account{},
					nil,
				)

				resp, err := plg.fetchNextUsers(ctx, req)

				Expect(err).To(BeNil())
				Expect(resp.Others).To(HaveLen(0))
				Expect(resp.HasMore).To(BeFalse())
			})
		})
	})
})
