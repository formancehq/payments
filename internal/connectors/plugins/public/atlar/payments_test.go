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
	"github.com/get-momo/atlar-v1-go-client/client/transactions"
	atlar_models "github.com/get-momo/atlar-v1-go-client/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/golang/mock/gomock"
)

var _ = Describe("Atlar Plugin Payments", func() {
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

	Context("fetching next payments", func() {
		var (
			samplePayments []*atlar_models.Transaction
			sampleAccount  atlar_models.Account
			now            time.Time
		)

		BeforeEach(func() {
			now = time.Now().UTC()

			sampleAccount = atlar_models.Account{
				Bank: &atlar_models.BankSlim{
					Bic: "test",
				},
				Bic:          "test",
				ThirdPartyID: "test",
			}

			samplePayments = make([]*atlar_models.Transaction, 0)
			for i := 0; i < 50; i++ {
				samplePayments = append(samplePayments, &atlar_models.Transaction{
					ID: fmt.Sprintf("%d", i),
					Amount: &atlar_models.Amount{
						Currency:    pointer.For("EUR"),
						StringValue: pointer.For("100"),
						Value:       pointer.For(int64(100)),
					},
					Account: &atlar_models.AccountTrx{
						ID: pointer.For("test"),
					},
					Characteristics: &atlar_models.TransactionCharacteristics{
						BankTransactionCode: &atlar_models.BankTransactionCode{
							Description: "test",
							Domain:      "test",
							Family:      "test",
							Subfamily:   "test",
						},
					},
					Created:     now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339Nano),
					Description: "test",
					Reconciliation: &atlar_models.ReconciliationDetails{
						BookedTransactionID:   "test",
						ExpectedTransactionID: "test",
						Status:                "test",
						TransactableID:        "test",
						TransactableType:      "test",
					},
					RemittanceInformation: &atlar_models.RemittanceInformation{
						Type:  pointer.For("test"),
						Value: pointer.For("test"),
					},
				})
			}
		})

		It("should return an error - get payments error", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				PageSize: 60,
			}

			m.EXPECT().GetV1Transactions(gomock.Any(), "", int64(60)).Return(
				&transactions.GetV1TransactionsOK{
					Payload: &transactions.GetV1TransactionsOKBody{
						QueryResponse: atlar_models.QueryResponse{
							NextToken: "",
						},
						Items: []*atlar_models.Transaction{},
					},
				},
				errors.New("test error"),
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should fetch next payments - no state no results", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				PageSize: 60,
			}

			m.EXPECT().GetV1Transactions(gomock.Any(), "", int64(60)).Return(
				&transactions.GetV1TransactionsOK{
					Payload: &transactions.GetV1TransactionsOKBody{
						QueryResponse: atlar_models.QueryResponse{
							NextToken: "",
						},
						Items: []*atlar_models.Transaction{},
					},
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.NextToken).To(BeEmpty())
		})

		It("should fetch next payments - no state pageSize > total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				PageSize: 60,
			}

			m.EXPECT().GetV1Transactions(gomock.Any(), "", int64(60)).Return(
				&transactions.GetV1TransactionsOK{
					Payload: &transactions.GetV1TransactionsOKBody{
						QueryResponse: atlar_models.QueryResponse{
							NextToken: "",
						},
						Items: samplePayments,
					},
				},
				nil,
			)

			for _, acc := range samplePayments {
				m.EXPECT().GetV1AccountsID(gomock.Any(), *acc.Account.ID).Return(
					&accounts.GetV1AccountsIDOK{
						Payload: &sampleAccount,
					},
					nil,
				)

				m.EXPECT().GetV1BetaThirdPartiesID(gomock.Any(), sampleAccount.ThirdPartyID).Return(
					&third_parties.GetV1betaThirdPartiesIDOK{
						Payload: &atlar_models.ThirdParty{
							ID:   "test",
							Name: "test",
						},
					},
					nil,
				)
			}

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(50))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.NextToken).To(BeEmpty())
		})

		It("should fetch next accounts - no state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				PageSize: 40,
			}

			m.EXPECT().GetV1Transactions(gomock.Any(), "", int64(40)).Return(
				&transactions.GetV1TransactionsOK{
					Payload: &transactions.GetV1TransactionsOKBody{
						QueryResponse: atlar_models.QueryResponse{
							NextToken: "1234",
						},
						Items: samplePayments[:40],
					},
				},
				nil,
			)

			for _, acc := range samplePayments[:40] {
				m.EXPECT().GetV1AccountsID(gomock.Any(), *acc.Account.ID).Return(
					&accounts.GetV1AccountsIDOK{
						Payload: &sampleAccount,
					},
					nil,
				)

				m.EXPECT().GetV1BetaThirdPartiesID(gomock.Any(), sampleAccount.ThirdPartyID).Return(
					&third_parties.GetV1betaThirdPartiesIDOK{
						Payload: &atlar_models.ThirdParty{
							ID:   "test",
							Name: "test",
						},
					},
					nil,
				)
			}

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(40))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.NextToken).To(Equal("1234"))
		})

		It("should fetch next accounts - with state pageSize < total accounts", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{"nextToken": "1234"}`),
				PageSize: 40,
			}

			m.EXPECT().GetV1Transactions(gomock.Any(), "1234", int64(40)).Return(
				&transactions.GetV1TransactionsOK{
					Payload: &transactions.GetV1TransactionsOKBody{
						QueryResponse: atlar_models.QueryResponse{
							NextToken: "",
						},
						Items: samplePayments[40:],
					},
				},
				nil,
			)

			for _, acc := range samplePayments[40:] {
				m.EXPECT().GetV1AccountsID(gomock.Any(), *acc.Account.ID).Return(
					&accounts.GetV1AccountsIDOK{
						Payload: &sampleAccount,
					},
					nil,
				)

				m.EXPECT().GetV1BetaThirdPartiesID(gomock.Any(), sampleAccount.ThirdPartyID).Return(
					&third_parties.GetV1betaThirdPartiesIDOK{
						Payload: &atlar_models.ThirdParty{
							ID:   "test",
							Name: "test",
						},
					},
					nil,
				)
			}

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(10))
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
