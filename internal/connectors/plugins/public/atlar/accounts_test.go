package atlar

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/atlar/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/get-momo/atlar-v1-go-client/client/accounts"
	"github.com/get-momo/atlar-v1-go-client/client/third_parties"
	atlar_models "github.com/get-momo/atlar-v1-go-client/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Atlar Plugin Accounts", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("fetching next accounts", func() {
		var (
			m              *client.MockClient
			sampleAccounts []*atlar_models.Account
			now            time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
			now = time.Now().UTC()

			sampleAccounts = make([]*atlar_models.Account, 0)
			for i := 0; i < 50; i++ {
				sampleAccounts = append(sampleAccounts, &atlar_models.Account{
					ID:           pointer.For(fmt.Sprintf("%d", i)),
					Created:      now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339Nano),
					Name:         fmt.Sprintf("Account %d", i),
					Currency:     "EUR",
					ThirdPartyID: fmt.Sprintf("t%d", i),
					Fictive:      false,
					Alias:        fmt.Sprintf("test-%d", i),
					Owner: &atlar_models.PartyIdentification{
						Name: fmt.Sprintf("owner-%d", i),
					},
					Bank: &atlar_models.BankSlim{
						Bic: "BIC",
					},
				})
			}
		})

		It("should return an error - get accounts error", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				PageSize: 60,
			}

			m.EXPECT().GetV1Accounts(gomock.Any(), "", int64(60)).Return(
				&accounts.GetV1AccountsOK{
					Payload: &accounts.GetV1AccountsOKBody{
						QueryResponse: atlar_models.QueryResponse{
							NextToken: "",
						},
						Items: sampleAccounts,
					},
				},
				errors.New("test error"),
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextAccountsResponse{}))
		})

		It("should fetch next accounts - no state no results", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				PageSize: 60,
			}

			m.EXPECT().GetV1Accounts(gomock.Any(), "", int64(60)).Return(
				&accounts.GetV1AccountsOK{
					Payload: &accounts.GetV1AccountsOKBody{
						QueryResponse: atlar_models.QueryResponse{
							NextToken: "",
						},
						Items: []*atlar_models.Account{},
					},
				},
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.NextToken).To(BeEmpty())
		})

		It("should fetch next accounts - no state pageSize > total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				PageSize: 60,
			}

			m.EXPECT().GetV1Accounts(gomock.Any(), "", int64(60)).Return(
				&accounts.GetV1AccountsOK{
					Payload: &accounts.GetV1AccountsOKBody{
						QueryResponse: atlar_models.QueryResponse{
							NextToken: "",
						},
						Items: sampleAccounts,
					},
				},
				nil,
			)

			for _, acc := range sampleAccounts {
				m.EXPECT().GetV1BetaThirdPartiesID(gomock.Any(), acc.ThirdPartyID).Return(
					&third_parties.GetV1betaThirdPartiesIDOK{
						Payload: &atlar_models.ThirdParty{
							ID:   "test",
							Name: "test",
						},
					},
					nil,
				)
			}

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(50))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.NextToken).To(BeEmpty())
		})

		It("should fetch next accounts - no state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				PageSize: 40,
			}

			m.EXPECT().GetV1Accounts(gomock.Any(), "", int64(40)).Return(
				&accounts.GetV1AccountsOK{
					Payload: &accounts.GetV1AccountsOKBody{
						QueryResponse: atlar_models.QueryResponse{
							NextToken: "1234",
						},
						Items: sampleAccounts[:40],
					},
				},
				nil,
			)

			for _, acc := range sampleAccounts[:40] {
				m.EXPECT().GetV1BetaThirdPartiesID(gomock.Any(), acc.ThirdPartyID).Return(
					&third_parties.GetV1betaThirdPartiesIDOK{
						Payload: &atlar_models.ThirdParty{
							ID:   "test",
							Name: "test",
						},
					},
					nil,
				)
			}

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(40))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.NextToken).To(Equal("1234"))
		})

		It("should fetch next accounts - with state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{"nextToken": "1234"}`),
				PageSize: 40,
			}

			m.EXPECT().GetV1Accounts(gomock.Any(), "1234", int64(40)).Return(
				&accounts.GetV1AccountsOK{
					Payload: &accounts.GetV1AccountsOKBody{
						QueryResponse: atlar_models.QueryResponse{
							NextToken: "",
						},
						Items: sampleAccounts[40:],
					},
				},
				nil,
			)

			for _, acc := range sampleAccounts[40:] {
				m.EXPECT().GetV1BetaThirdPartiesID(gomock.Any(), acc.ThirdPartyID).Return(
					&third_parties.GetV1betaThirdPartiesIDOK{
						Payload: &atlar_models.ThirdParty{
							ID:   "test",
							Name: "test",
						},
					},
					nil,
				)
			}

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(10))
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
