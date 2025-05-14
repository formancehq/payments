package atlar

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/atlar/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/get-momo/atlar-v1-go-client/client/counterparties"
	"github.com/get-momo/atlar-v1-go-client/client/external_accounts"
	atlar_models "github.com/get-momo/atlar-v1-go-client/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Atlar Plugin External Accounts", func() {
	var (
		m   *client.MockClient
		plg models.Plugin
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m}
	})

	Context("fetching next external accounts", func() {
		var (
			sampleAccounts []*atlar_models.ExternalAccount
			now            time.Time
		)

		BeforeEach(func() {
			now = time.Now().UTC()

			sampleAccounts = make([]*atlar_models.ExternalAccount, 0)
			for i := 0; i < 50; i++ {
				sampleAccounts = append(sampleAccounts, &atlar_models.ExternalAccount{
					ID:      fmt.Sprintf("%d", i),
					Created: now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339Nano),
					Bank: &atlar_models.BankSlim{
						Bic:  "1234",
						ID:   "testBank",
						Name: "testBank",
					},
					CounterpartyID: "1234",
				})
			}
		})

		It("should return an error - get external accounts error", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				PageSize: 60,
			}

			m.EXPECT().GetV1ExternalAccounts(gomock.Any(), "", int64(60)).Return(
				&external_accounts.GetV1ExternalAccountsOK{
					Payload: &external_accounts.GetV1ExternalAccountsOKBody{
						QueryResponse: atlar_models.QueryResponse{
							NextToken: "",
						},
						Items: []*atlar_models.ExternalAccount{},
					},
				},
				errors.New("test error"),
			)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextExternalAccountsResponse{}))
		})

		It("should fetch next external accounts - no state no results", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				PageSize: 60,
			}

			m.EXPECT().GetV1ExternalAccounts(gomock.Any(), "", int64(60)).Return(
				&external_accounts.GetV1ExternalAccountsOK{
					Payload: &external_accounts.GetV1ExternalAccountsOKBody{
						QueryResponse: atlar_models.QueryResponse{
							NextToken: "",
						},
						Items: []*atlar_models.ExternalAccount{},
					},
				},
				nil,
			)

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.NextToken).To(BeEmpty())
		})

		It("should fetch next external accounts - no state pageSize > total accounts", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				PageSize: 60,
			}

			m.EXPECT().GetV1ExternalAccounts(gomock.Any(), "", int64(60)).Return(
				&external_accounts.GetV1ExternalAccountsOK{
					Payload: &external_accounts.GetV1ExternalAccountsOKBody{
						QueryResponse: atlar_models.QueryResponse{
							NextToken: "",
						},
						Items: sampleAccounts,
					},
				},
				nil,
			)

			for _, acc := range sampleAccounts {
				m.EXPECT().GetV1CounterpartiesID(gomock.Any(), acc.CounterpartyID).Return(
					&counterparties.GetV1CounterpartiesIDOK{
						Payload: &atlar_models.Counterparty{
							ContactDetails: &atlar_models.ContactDetails{
								Address: &atlar_models.Address{
									City:         "Paris",
									Country:      "France",
									PostalCode:   "75013",
									StreetName:   "test",
									StreetNumber: "42",
								},
								Email:      "test@test.com",
								NationalID: "1",
							},
							Created:    pointer.For(now.Format(time.RFC3339Nano)),
							ExternalID: "123",
							ID:         pointer.For("123"),
							Name:       "test",
						},
					},
					nil,
				)
			}

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(50))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.NextToken).To(BeEmpty())
		})

		It("should fetch next external accounts - no state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				PageSize: 40,
			}

			m.EXPECT().GetV1ExternalAccounts(gomock.Any(), "", int64(40)).Return(
				&external_accounts.GetV1ExternalAccountsOK{
					Payload: &external_accounts.GetV1ExternalAccountsOKBody{
						QueryResponse: atlar_models.QueryResponse{
							NextToken: "1234",
						},
						Items: sampleAccounts[:40],
					},
				},
				nil,
			)

			for _, acc := range sampleAccounts[:40] {
				m.EXPECT().GetV1CounterpartiesID(gomock.Any(), acc.CounterpartyID).Return(
					&counterparties.GetV1CounterpartiesIDOK{
						Payload: &atlar_models.Counterparty{
							ContactDetails: &atlar_models.ContactDetails{
								Address: &atlar_models.Address{
									City:         "Paris",
									Country:      "France",
									PostalCode:   "75013",
									StreetName:   "test",
									StreetNumber: "42",
								},
								Email:      "test@test.com",
								NationalID: "1",
							},
							Created:    pointer.For(now.Format(time.RFC3339Nano)),
							ExternalID: "123",
							ID:         pointer.For("123"),
							Name:       "test",
						},
					},
					nil,
				)
			}

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(40))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.NextToken).To(Equal("1234"))
		})

		It("should fetch next external accounts - with state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:    []byte(`{"nextToken": "1234"}`),
				PageSize: 40,
			}

			m.EXPECT().GetV1ExternalAccounts(gomock.Any(), "1234", int64(40)).Return(
				&external_accounts.GetV1ExternalAccountsOK{
					Payload: &external_accounts.GetV1ExternalAccountsOKBody{
						QueryResponse: atlar_models.QueryResponse{
							NextToken: "",
						},
						Items: sampleAccounts[40:],
					},
				},
				nil,
			)

			for _, acc := range sampleAccounts[40:] {
				m.EXPECT().GetV1CounterpartiesID(gomock.Any(), acc.CounterpartyID).Return(
					&counterparties.GetV1CounterpartiesIDOK{
						Payload: &atlar_models.Counterparty{
							ContactDetails: &atlar_models.ContactDetails{
								Address: &atlar_models.Address{
									City:         "Paris",
									Country:      "France",
									PostalCode:   "75013",
									StreetName:   "test",
									StreetNumber: "42",
								},
								Email:      "test@test.com",
								NationalID: "1",
							},
							Created:    pointer.For(now.Format(time.RFC3339Nano)),
							ExternalID: "123",
							ID:         pointer.For("123"),
							Name:       "test",
						},
					},
					nil,
				)
			}

			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(10))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.NextToken).To(BeEmpty())
		})
	})
})
