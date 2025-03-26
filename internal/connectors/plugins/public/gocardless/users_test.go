package gocardless

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/gocardless/client"
	"github.com/formancehq/payments/internal/models"
	gocardless "github.com/gocardless/gocardless-pro-go/v4"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Gocardless Plugin fetch next users", func() {
	var (
		plg         *Plugin
		now         time.Time
		sampleUsers []client.GocardlessUser
	)

	BeforeEach(func() {
		plg = &Plugin{}
		sampleUsers = make([]client.GocardlessUser, 0)
		for i := 0; i < 50; i++ {
			sampleUsers = append(sampleUsers, client.GocardlessUser{
				Id:          fmt.Sprintf("%d", i),
				Name:        fmt.Sprintf("Creditor %d", i),
				CountryCode: "US",
				CreatedAt:   now,
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
			now = time.Now().UTC()

		})

		It("should return an error - get users error", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{
				Name:     fetchOthers,
				PageSize: 1,
			}

			m.EXPECT().GetCreditors(gomock.Any(), 1, "CR123").Return(
				[]client.GocardlessUser{},
				client.Cursor{},
				errors.New("test error"),
			)

			m.EXPECT().GetCustomers(gomock.Any(), 1, "CU123").Return(
				[]client.GocardlessUser{},
				client.Cursor{},
				errors.New("test error"),
			)

			creditorResp, _, creditorErr := plg.client.GetCreditors(ctx, req.PageSize, "CR123")
			customerResp, _, customerErr := plg.client.GetCustomers(ctx, req.PageSize, "CU123")

			Expect(creditorErr).To(MatchError("test error"))
			Expect(customerErr).To(MatchError("test error"))

			Expect(creditorResp).To(Equal([]client.GocardlessUser{}))
			Expect(customerResp).To(Equal([]client.GocardlessUser{}))

		})

		It("should fetch next users - no results", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{
				Name:     fetchOthers,
				PageSize: 60,
			}

			m.EXPECT().GetCreditors(gomock.Any(), req.PageSize, "").Return(
				[]client.GocardlessUser{},
				client.Cursor{},
				nil,
			)

			m.EXPECT().GetCustomers(gomock.Any(), req.PageSize, "").Return(
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

			Expect(state.CreditorsAfter).To(Equal(""))
			Expect(state.CustomersAfter).To(Equal(""))

		})

		It("should fetch next users - pageSize > total users", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{
				Name:     fetchOthers,
				PageSize: 60,
				State:    json.RawMessage(`{"creditorsAfter": "CR123", "customersAfter": "CU123"}`),
			}

			m.EXPECT().GetCreditors(gomock.Any(), req.PageSize, "CR123").Return(
				sampleUsers,
				client.Cursor{
					After: "",
				},
				nil,
			)

			m.EXPECT().GetCustomers(gomock.Any(), req.PageSize, "CU123").Return(
				sampleUsers,
				client.Cursor{
					After: "",
				},
				nil,
			)

			resp, err := plg.FetchNextOthers(ctx, req)

			Expect(err).To(BeNil())
			Expect(resp.Others).To(HaveLen(100))
			Expect(resp.Others[0].ID).To(Equal("0"))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state usersState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())

		})

		It("should fetch users - pageSize < total users", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{
				Name:     fetchOthers,
				PageSize: 40,
			}

			m.EXPECT().GetCreditors(gomock.Any(), req.PageSize, "").Return(
				sampleUsers[:40],
				client.Cursor{},
				nil,
			)

			m.EXPECT().GetCustomers(gomock.Any(), req.PageSize, "").Return(
				sampleUsers[:40],
				client.Cursor{},
				nil,
			)

			resp, err := plg.FetchNextOthers(ctx, req)

			Expect(err).To(BeNil())
			Expect(resp.Others).To(HaveLen(80))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state usersState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())

		})

		It("should fetch next users - with state pageSize < total users", func(ctx SpecContext) {

			req := models.FetchNextOthersRequest{
				Name:     fetchOthers,
				PageSize: 40,
				State: json.RawMessage(`{
				"creditorsAfter": "CR1",
				"customersAfter": "CU1"
				}`),
			}

			m.EXPECT().GetCreditors(gomock.Any(), req.PageSize, "CR1").Return(
				sampleUsers[:40],
				client.Cursor{
					After: "",
				},
				nil,
			)

			m.EXPECT().GetCustomers(gomock.Any(), req.PageSize, "CU1").Return(
				sampleUsers[:40],
				client.Cursor{
					After: "",
				},
				nil,
			)

			resp, err := plg.FetchNextOthers(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Others).To(HaveLen(80))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state usersState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())

		})
	})

	Context("create bank account with gocardless client", func() {
		var (
			mockedService *client.MockGoCardlessService
		)

		BeforeEach((func() {
			ctrl := gomock.NewController(GinkgoT())
			mockedService = client.NewMockGoCardlessService(ctrl)

			plg.client, _ = client.New("test", "https://example.com", "access_token", true)
			plg.client.NewWithService(mockedService)
			now = time.Now().UTC()

		}))

		It("should fail when gocardless creditors client returns an error", func(ctx SpecContext) {

			req := models.FetchNextOthersRequest{
				Name:     fetchOthers,
				PageSize: 40,
			}

			mockedService.EXPECT().GetGocardlessCreditors(gomock.Any(), gomock.Any()).Return(
				nil,
				errors.New("fetch creditors error"),
			)

			resp, _, err := plg.client.GetCreditors(ctx, req.PageSize, "")

			Expect(err).To(MatchError("fetch creditors error"))
			Expect(resp).To(Equal([]client.GocardlessUser{}))

		})

		It("should fail when gocardless customers client returns an error", func(ctx SpecContext) {

			req := models.FetchNextOthersRequest{
				Name:     fetchOthers,
				PageSize: 40,
			}

			mockedService.EXPECT().GetGocardlessCustomers(gomock.Any(), gomock.Any()).Return(
				nil,
				errors.New("fetch customers error"),
			)

			resp, _, err := plg.client.GetCustomers(ctx, req.PageSize, "")

			Expect(err).To(MatchError("fetch customers error"))
			Expect(resp).To(Equal([]client.GocardlessUser{}))

		})

		It("should return error when invalid creditor CreatedAt is parsed", func(ctx SpecContext) {

			req := models.FetchNextOthersRequest{
				Name:     fetchOthers,
				PageSize: 1,
			}

			mockedService.EXPECT().GetGocardlessCreditors(gomock.Any(), gomock.Any()).Return(
				&gocardless.CreditorListResult{
					Creditors: []gocardless.Creditor{{
						CreatedAt: "invalid-time",
					}},
				},
				nil,
			)

			resp, _, err := plg.client.GetCreditors(ctx, req.PageSize, "")

			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to parse creation time"))
			Expect(resp).To(Equal([]client.GocardlessUser{}))
		})

		It("should return error when invalid customer CreatedAt is parsed", func(ctx SpecContext) {

			req := models.FetchNextOthersRequest{
				Name:     fetchOthers,
				PageSize: 1,
			}

			mockedService.EXPECT().GetGocardlessCustomers(gomock.Any(), gomock.Any()).Return(
				&gocardless.CustomerListResult{
					Customers: []gocardless.Customer{{
						CreatedAt: "invalid-time",
					}},
				},
				nil,
			)

			resp, _, err := plg.client.GetCustomers(ctx, req.PageSize, "")

			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to parse creation time"))
			Expect(resp).To(Equal([]client.GocardlessUser{}))
		})

		It("should return GocardlessUser when valid creditor CreatedAt is parsed", func(ctx SpecContext) {

			req := models.FetchNextOthersRequest{
				Name:     fetchOthers,
				PageSize: 1,
			}

			mockedService.EXPECT().GetGocardlessCreditors(gomock.Any(), gomock.Any()).Return(
				&gocardless.CreditorListResult{
					Creditors: []gocardless.Creditor{{
						CreatedAt: "2025-02-23T14:30:15.123456789Z",
						Id:        "CR123",
						Name:      "Creditor 123",
					}},
					Meta: gocardless.CreditorListResultMeta{
						Cursors: &gocardless.CreditorListResultMetaCursors{
							After: "CR123",
						},
					},
				},
				nil,
			)

			resp, _, err := plg.client.GetCreditors(ctx, req.PageSize, "")

			Expect(err).To(BeNil())
			Expect(resp).To(HaveLen(1))
			Expect(resp[0].Id).To(Equal("CR123"))
			Expect(resp[0].Name).To(Equal("Creditor 123"))

		})

		It("should return GocardlessUser when valid customer CreatedAt is parsed", func(ctx SpecContext) {

			req := models.FetchNextOthersRequest{
				Name:     fetchOthers,
				PageSize: 1,
			}

			mockedService.EXPECT().GetGocardlessCustomers(gomock.Any(), gomock.Any()).Return(
				&gocardless.CustomerListResult{
					Customers: []gocardless.Customer{{
						CreatedAt:  "2025-02-23T14:30:15.123456789Z",
						Id:         "CU123",
						GivenName:  "John",
						FamilyName: "Doe",
					}},
					Meta: gocardless.CustomerListResultMeta{
						Cursors: &gocardless.CustomerListResultMetaCursors{
							After: "CU123",
						},
					},
				},
				nil,
			)

			resp, _, err := plg.client.GetCustomers(ctx, req.PageSize, "")

			Expect(err).To(BeNil())
			Expect(resp).To(HaveLen(1))
			Expect(resp[0].Id).To(Equal("CU123"))
			Expect(resp[0].Name).To(Equal("John Doe"))
		})
	})
})
