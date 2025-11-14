package mangopay

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/mangopay/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/golang/mock/gomock"
)

var _ = Describe("Mangopay Plugin Users", func() {
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

	Context("fetching next users", func() {
		var (
			sampleUsers []client.User
			now         time.Time
		)

		BeforeEach(func() {
			now = time.Now().UTC()

			sampleUsers = make([]client.User, 0)
			for i := 0; i < 50; i++ {
				sampleUsers = append(sampleUsers, client.User{
					ID:           fmt.Sprintf("%d", i),
					CreationDate: now.Add(-time.Duration(50-i) * time.Minute).UTC().Unix(),
				})
			}
		})

		It("should return an error - get users error", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{
				Name:     fetchUsersName,
				PageSize: 60,
			}

			m.EXPECT().GetUsers(gomock.Any(), 1, 60).Return(
				[]client.User{},
				errors.New("test error"),
			)

			resp, err := plg.FetchNextOthers(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextOthersResponse{}))
		})

		It("should fetch next users - no state no results", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{
				Name:     fetchUsersName,
				PageSize: 60,
			}

			m.EXPECT().GetUsers(gomock.Any(), 1, 60).Return(
				[]client.User{},
				nil,
			)

			resp, err := plg.FetchNextOthers(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Others).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state usersState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.LastPage).To(Equal(1))
			Expect(state.LastCreationDate.IsZero()).To(BeTrue())
		})

		It("should fetch next users - no state pageSize > total users", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{
				Name:     fetchUsersName,
				PageSize: 60,
			}

			m.EXPECT().GetUsers(gomock.Any(), 1, 60).Return(
				sampleUsers,
				nil,
			)

			resp, err := plg.FetchNextOthers(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Others).To(HaveLen(50))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state usersState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			createdTime := time.Unix(sampleUsers[49].CreationDate, 0)
			Expect(state.LastPage).To(Equal(1))
			Expect(state.LastCreationDate.UTC()).To(Equal(createdTime.UTC()))
		})

		It("should fetch users - no state pageSize < total users", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{
				Name:     fetchUsersName,
				PageSize: 40,
			}

			m.EXPECT().GetUsers(gomock.Any(), 1, 40).Return(
				sampleUsers[:40],
				nil,
			)

			resp, err := plg.FetchNextOthers(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Others).To(HaveLen(40))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())

			var state usersState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastPage).To(Equal(1))
			createdTime := time.Unix(sampleUsers[39].CreationDate, 0)
			Expect(state.LastCreationDate.UTC()).To(Equal(createdTime.UTC()))
		})

		It("should fetch next users - with state pageSize < total users", func(ctx SpecContext) {
			lastCreatedAt := time.Unix(sampleUsers[39].CreationDate, 0)
			req := models.FetchNextOthersRequest{
				Name:     fetchUsersName,
				State:    []byte(fmt.Sprintf(`{"lastPage": 1, "lastCreationDate": "%s"}`, lastCreatedAt.UTC().Format(time.RFC3339Nano))),
				PageSize: 40,
			}

			m.EXPECT().GetUsers(gomock.Any(), 1, 40).Return(
				sampleUsers[:40],
				nil,
			)

			m.EXPECT().GetUsers(gomock.Any(), 2, 40).Return(
				sampleUsers[41:],
				nil,
			)

			resp, err := plg.FetchNextOthers(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Others).To(HaveLen(10))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state usersState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.LastPage).To(Equal(2))
			createdTime := time.Unix(sampleUsers[49].CreationDate, 0)
			Expect(state.LastCreationDate.UTC()).To(Equal(createdTime.UTC()))
		})

		It("should fetch next users - when lastCreationDate form last page is equal to one of the new page's", func(ctx SpecContext) {
			lastCreatedAt := time.Unix(sampleUsers[9].CreationDate, 0)
			sampleUsers[10].CreationDate = sampleUsers[9].CreationDate
			req := models.FetchNextOthersRequest{
				Name:     fetchUsersName,
				State:    []byte(fmt.Sprintf(`{"lastPage": 2, "lastCreationDate": "%s"}`, lastCreatedAt.UTC().Format(time.RFC3339Nano))),
				PageSize: 10,
			}

			m.EXPECT().GetUsers(gomock.Any(), 2, 10).Times(1).Return(
				sampleUsers[10:20],
				nil,
			)

			resp, err := plg.FetchNextOthers(ctx, req)

			Expect(err).To(BeNil())
			Expect(resp.Others).To(HaveLen(10))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())

			var state usersState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastPage).To(Equal(2))
			createdTime := time.Unix(sampleUsers[19].CreationDate, 0)
			Expect(state.LastCreationDate.UTC()).To(Equal(createdTime.UTC()))
		})
	})
})
