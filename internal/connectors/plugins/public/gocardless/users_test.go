package gocardless

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/gocardless/client"
	"github.com/formancehq/payments/internal/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Gocardless Plugin fetch next users", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("fetching next users", func() {
		var (
			m           *client.MockClient
			sampleUsers []client.GocardlessUser
			now         time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
			now = time.Now().UTC()

			sampleUsers = make([]client.GocardlessUser, 0)
			for i := 0; i < 50; i++ {
				sampleUsers = append(sampleUsers, client.GocardlessUser{
					Id:          fmt.Sprintf("%d", i),
					Name:        fmt.Sprintf("Creditor %d", i),
					CountryCode: "US",
					CreatedAt:   now.Add(-time.Duration(50-i) * time.Minute).Unix(),
				})
			}

		})

		It("should return an error - get users error", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{
				Name:     fetchOthers,
				PageSize: 1,
			}

			m.EXPECT().GetCreditors(gomock.Any(), 1, "CR123", "CR124").Return(
				[]client.GocardlessUser{},
				client.Cursor{},
				errors.New("test error"),
			)

			m.EXPECT().GetCustomers(gomock.Any(), 1, "CR123", "CR124").Return(
				[]client.GocardlessUser{},
				client.Cursor{},
				errors.New("test error"),
			)

			creditorResp, _, creditorErr := plg.client.GetCreditors(ctx, req.PageSize, "CR123", "CR124")
			customerResp, _, customerErr := plg.client.GetCustomers(ctx, req.PageSize, "CR123", "CR124")

			Expect(creditorErr).To(MatchError("test error"))
			Expect(customerErr).To(MatchError("test error"))

			Expect(creditorResp).To(Equal([]client.GocardlessUser{}))
			Expect(customerResp).To(Equal([]client.GocardlessUser{}))

		})

		It("should fetch next users - no state no results", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{
				Name:        fetchOthers,
				PageSize:    60,
				FromPayload: json.RawMessage(`{"reference": "CR123"}`),
			}

			m.EXPECT().GetCreditors(gomock.Any(), req.PageSize, "", "").Return(
				[]client.GocardlessUser{},
				client.Cursor{},
				nil,
			)

			m.EXPECT().GetCustomers(gomock.Any(), req.PageSize, "", "").Return(
				[]client.GocardlessUser{},
				client.Cursor{},
				nil,
			)

			creditorResp, creditorErr := plg.FetchNextOthers(ctx, req)

			Expect(creditorErr).To(BeNil())
			Expect(creditorResp.Others).To(HaveLen(0))
			Expect(creditorResp.HasMore).To(BeFalse())
			Expect(creditorResp.NewState).ToNot(BeNil())

			var state usersState
			creditorErr = json.Unmarshal(creditorResp.NewState, &state)
			Expect(creditorErr).To(BeNil())

			Expect(state.After).To(Equal(""))
			Expect(state.Before).To(Equal(""))

			req.FromPayload = json.RawMessage(`{"reference": "CU123"}`)
			customerResp, customerErr := plg.FetchNextOthers(ctx, req)

			Expect(customerErr).To(BeNil())
			Expect(customerResp.Others).To(HaveLen(0))
			Expect(customerResp.HasMore).To(BeFalse())
			Expect(customerResp.NewState).ToNot(BeNil())

			var customerState usersState
			customerErr = json.Unmarshal(customerResp.NewState, &customerState)
			Expect(customerErr).To(BeNil())

			Expect(customerState.After).To(Equal(""))
			Expect(customerState.Before).To(Equal(""))
		})

		It("should fetch next users - no state pageSize > total users", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{
				Name:        fetchOthers,
				PageSize:    60,
				FromPayload: json.RawMessage(`{"reference": "CR123"}`),
				State:       json.RawMessage(`{"after": "CR123", "before": "CR124"}`),
			}

			m.EXPECT().GetCreditors(gomock.Any(), req.PageSize, "CR123", "CR124").Return(
				sampleUsers,
				client.Cursor{
					After:  "CR123",
					Before: "CR124",
				},
				nil,
			)

			resp, err := plg.FetchNextOthers(ctx, req)

			Expect(err).To(BeNil())
			Expect(resp.Others).To(HaveLen(50))
			Expect(resp.Others[0].ID).To(Equal("0"))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state usersState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())

			Expect(state.After).To(Equal("CR123"))
			Expect(state.Before).To(Equal("CR124"))

			// We fetched everything, state should be resetted
			createdTime := time.Unix(sampleUsers[49].CreatedAt, 0)
			Expect(state.LastCreationDate.UTC()).To(Equal(createdTime.UTC()))
		})

		It("should fetch users - no state pageSize < total users", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{
				Name:        fetchOthers,
				PageSize:    40,
				FromPayload: json.RawMessage(`{"reference": "CR123"}`),
			}

			m.EXPECT().GetCreditors(gomock.Any(), req.PageSize, "", "").Return(
				sampleUsers[:40],
				client.Cursor{},
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

			createdTime := time.Unix(sampleUsers[39].CreatedAt, 0)
			Expect(state.LastCreationDate.UTC()).To(Equal(createdTime.UTC()))

		})

		It("should fetch next users - with state pageSize < total users", func(ctx SpecContext) {
			lastCreationDate := time.Unix(sampleUsers[38].CreatedAt, 0)

			req := models.FetchNextOthersRequest{
				Name:        fetchOthers,
				PageSize:    40,
				FromPayload: json.RawMessage(`{"reference": "CR123"}`),
				State:       json.RawMessage(`{"after": "CR1", "before": "CR40",  "lastCreationDate": "` + lastCreationDate.Format(time.RFC3339Nano) + `"}`),
			}

			m.EXPECT().GetCreditors(gomock.Any(), req.PageSize, "CR1", "CR40").Return(
				sampleUsers[:40],
				client.Cursor{
					After:  "CR40",
					Before: "CR1",
				},
				nil,
			)

			m.EXPECT().GetCreditors(gomock.Any(), req.PageSize, "CR40", "CR1").Return(
				sampleUsers[41:],
				client.Cursor{
					After:  "CR50",
					Before: "CR41",
				},
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

			Expect(state.After).To(Equal("CR50"))
			Expect(state.Before).To(Equal("CR41"))
		})
	})
})
