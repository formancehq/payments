package gocardless

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/gocardless/client"
	"github.com/formancehq/payments/internal/models"
	gocardless "github.com/gocardless/gocardless-pro-go/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Gocardless Plugin fetch next external accounts", func() {
	var (
		plg                *Plugin
		sampleBankAccounts []client.GocardlessGenericAccount
		now                time.Time
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("when there are no external accounts", func() {
		var (
			m *client.MockClient
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
			now = time.Now().UTC()

			sampleBankAccounts = make([]client.GocardlessGenericAccount, 0)
			for i := 0; i < 50; i++ {
				name := fmt.Sprintf("Account %d", i)
				sampleBankAccounts = append(sampleBankAccounts, client.GocardlessGenericAccount{
					ID:                fmt.Sprintf("BA%d", i),
					AccountHolderName: name,
					CreatedAt:         now,
					Metadata:          map[string]interface{}{"type": "external_account"},
					Currency:          "USD",
					AccountType:       "savings",
				})
			}
		})

		It("should return an error - get beneficiaries error", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				FromPayload: json.RawMessage(`{"id": "CR123"}`),
				PageSize:    60,
			}

			m.EXPECT().GetExternalAccounts(gomock.Any(), "CR123", 60, "").
				Return(sampleBankAccounts, client.Cursor{}, errors.New("get beneficiaries error"))

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("get beneficiaries error"))
			Expect(resp).To(Equal(models.FetchNextExternalAccountsResponse{}))
		})

		It("should return an error when FromPayload is nill", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				PageSize: 60,
			}

			resp, err := plg.fetchNextExternalAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing from payload in request"))
			Expect(resp).To(Equal(models.FetchNextExternalAccountsResponse{}))

		})

		It("should return an error when id is empty", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				FromPayload: json.RawMessage(`{"id": ""}`),
				PageSize:    60,
			}

			resp, err := plg.fetchNextExternalAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("id field is required"))
			Expect(resp).To(Equal(models.FetchNextExternalAccountsResponse{}))

		})

		It("should return an error when id is empty", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				FromPayload: json.RawMessage(`{"id": "ZXY"}`),
				PageSize:    60,
			}

			resp, err := plg.fetchNextExternalAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("ownerId field must start with 'CR' for creditor account or 'CU' customer account"))
			Expect(resp).To(Equal(models.FetchNextExternalAccountsResponse{}))

		})

		It("should fetch next external customers accounts - no state no results", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				FromPayload: json.RawMessage(`{"id": "CU123"}`),
				PageSize:    60,
			}

			m.EXPECT().GetExternalAccounts(gomock.Any(), "CU123", req.PageSize, "").
				Return(
					[]client.GocardlessGenericAccount{},
					client.Cursor{},
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
			Expect(state.After).To(Equal(""))
		})

		It("should fetch next external customers accounts - no state pageSize > total accounts", func(ctx SpecContext) {

			req := models.FetchNextExternalAccountsRequest{
				FromPayload: json.RawMessage(`{"id": "CU123"}`),
				PageSize:    60,
			}

			m.EXPECT().GetExternalAccounts(gomock.Any(), "CU123", req.PageSize, "").
				Return(sampleBankAccounts, client.Cursor{
					After: "",
				}, nil)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(50))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state externalAccountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.After).To(Equal(sampleBankAccounts[len(sampleBankAccounts)-1].ID))

		})

		It("should return an error when marshalling failed", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				FromPayload: json.RawMessage(`{"id": "CU123"}`),
				PageSize:    60,
			}

			m.EXPECT().GetExternalAccounts(gomock.Any(), "CU123", req.PageSize, "").
				Return(
					[]client.GocardlessGenericAccount{},
					client.Cursor{},
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
			Expect(state.After).To(Equal(""))
		})

		It("should fetch next external creditors accounts - no state pageSize > total accounts", func(ctx SpecContext) {

			req := models.FetchNextExternalAccountsRequest{
				FromPayload: json.RawMessage(`{"id": "CR123"}`),
				PageSize:    60,
			}

			m.EXPECT().GetExternalAccounts(gomock.Any(), "CR123", req.PageSize, "").
				Return(sampleBankAccounts, client.Cursor{
					After: "",
				}, nil)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(50))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state externalAccountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.After).To(Equal(sampleBankAccounts[len(sampleBankAccounts)-1].ID))

		})

		It("should fetch next creditors accounts - no state pageSize < total accounts", func(ctx SpecContext) {

			req := models.FetchNextExternalAccountsRequest{
				FromPayload: json.RawMessage(`{"id": "CR123"}`),
				PageSize:    40,
			}

			m.EXPECT().GetExternalAccounts(gomock.Any(), "CR123", req.PageSize, "").
				Return(sampleBankAccounts[:40], client.Cursor{
					After: "BA40",
				}, nil)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(40))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())

			var state externalAccountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.After).To(Equal("BA40"))
		})

		It("should fetch next external creditors accounts - with state pageSize < total accounts", func(ctx SpecContext) {

			req := models.FetchNextExternalAccountsRequest{
				FromPayload: json.RawMessage(`{"id": "CR123"}`),
				PageSize:    40,
				State:       []byte(`{"after": "BA41" }`),
			}

			m.EXPECT().GetExternalAccounts(gomock.Any(), "CR123", req.PageSize, "BA41").
				Return(sampleBankAccounts[41:], client.Cursor{
					After: "",
				}, nil)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(9))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state externalAccountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.After).To(Equal(sampleBankAccounts[41:][len(sampleBankAccounts[41:])-1].ID))

		})

	})

	Context("fetch next external accounts with gocardless client", func() {
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

		It("should fail when fetching gocardless creditor's accounts returns an error", func(ctx SpecContext) {

			req := models.FetchNextExternalAccountsRequest{
				FromPayload: json.RawMessage(`{"id": "CR123"}`),
				PageSize:    40,
			}

			mockedService.EXPECT().GetGocardlessCreditorBankAccounts(gomock.Any(), gomock.Any()).Return(
				nil,
				errors.New("fetch creditors accounts error"),
			)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)

			Expect(err).To(MatchError("fetch creditors accounts error"))
			Expect(resp).To(Equal(models.FetchNextExternalAccountsResponse{}))

		})

		It("should fail when fetching gocardless customer's accounts returns an error", func(ctx SpecContext) {

			req := models.FetchNextExternalAccountsRequest{
				FromPayload: json.RawMessage(`{"id": "CU123"}`),
				PageSize:    40,
			}

			mockedService.EXPECT().GetGocardlessCustomerBankAccounts(gomock.Any(), gomock.Any()).Return(
				nil,
				errors.New("fetch customers accounts error"),
			)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)

			Expect(err).To(MatchError("fetch customers accounts error"))
			Expect(resp).To(Equal(models.FetchNextExternalAccountsResponse{}))

		})

		It("should fail when fetching gocardless creditors accounts returns an invalid created at", func(ctx SpecContext) {

			req := models.FetchNextExternalAccountsRequest{
				FromPayload: json.RawMessage(`{"id": "CR123"}`),
				PageSize:    40,
			}

			mockedService.EXPECT().GetGocardlessCreditorBankAccounts(gomock.Any(), gomock.Any()).Return(
				&gocardless.CreditorBankAccountListResult{
					CreditorBankAccounts: []gocardless.CreditorBankAccount{{
						CreatedAt: "invalid-date",
					}},
				},
				nil,
			)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)

			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to parse creation time"))
			Expect(resp).To(Equal(models.FetchNextExternalAccountsResponse{}))

		})

		It("should fail when fetching gocardless customers accounts returns an invalid created at", func(ctx SpecContext) {

			req := models.FetchNextExternalAccountsRequest{
				FromPayload: json.RawMessage(`{"id": "CU123"}`),
				PageSize:    40,
			}

			mockedService.EXPECT().GetGocardlessCustomerBankAccounts(gomock.Any(), gomock.Any()).Return(
				&gocardless.CustomerBankAccountListResult{
					CustomerBankAccounts: []gocardless.CustomerBankAccount{{
						CreatedAt: "invalid-date",
					}},
				},
				nil,
			)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)

			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to parse creation time"))
			Expect(resp).To(Equal(models.FetchNextExternalAccountsResponse{}))

		})

		It("should return results when fetching gocardless creditors accounts returns a valid response", func(ctx SpecContext) {

			req := models.FetchNextExternalAccountsRequest{
				FromPayload: json.RawMessage(`{"id": "CR123"}`),
				PageSize:    40,
			}

			accountName := pointer.For("test-account")

			mockedService.EXPECT().GetGocardlessCreditorBankAccounts(gomock.Any(), gomock.Any()).Return(
				&gocardless.CreditorBankAccountListResult{
					CreditorBankAccounts: []gocardless.CreditorBankAccount{{
						CreatedAt:         "2025-02-23T14:30:15.123456789Z",
						AccountHolderName: "test-account",
						BankName:          "test-bank",
						Currency:          "USD",
						Metadata:          map[string]interface{}{},
					}},
					Meta: gocardless.CreditorBankAccountListResultMeta{
						Cursors: &gocardless.CreditorBankAccountListResultMetaCursors{
							After: "",
						},
					},
				},
				nil,
			)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)

			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(1))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.ExternalAccounts[0].Name).To(Equal(accountName))

		})

		It("should return results when fetching gocardless customers accounts returns a valid response", func(ctx SpecContext) {

			req := models.FetchNextExternalAccountsRequest{
				FromPayload: json.RawMessage(`{"id": "CU123"}`),
				PageSize:    40,
			}

			accountName := pointer.For("test-account")

			mockedService.EXPECT().GetGocardlessCustomerBankAccounts(gomock.Any(), gomock.Any()).Return(
				&gocardless.CustomerBankAccountListResult{
					CustomerBankAccounts: []gocardless.CustomerBankAccount{{
						CreatedAt:         "2025-02-23T14:30:15.123456789Z",
						AccountHolderName: "test-account",
						BankName:          "test-bank",
						Currency:          "USD",
						Metadata:          map[string]interface{}{},
					}},
					Meta: gocardless.CustomerBankAccountListResultMeta{
						Cursors: &gocardless.CustomerBankAccountListResultMetaCursors{
							After: "",
						},
					},
				},
				nil,
			)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)

			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(1))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.ExternalAccounts[0].Name).To(Equal(accountName))

		})
	})
})
