package plaid

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/plaid/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/plaid/plaid-go/v34/plaid"
	gomock "github.com/golang/mock/gomock"
)

var _ = Describe("Plaid *Plugin Accounts", func() {
	Context("fetch next accounts", func() {
		var (
			ctrl *gomock.Controller
			plg  models.Plugin
			m    *client.MockClient

			sampleAccounts []plaid.AccountBase
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{client: m}

			sampleAccounts = make([]plaid.AccountBase, 0)
			for i := 0; i < 3; i++ {
				account := plaid.NewAccountBaseWithDefaults()
				account.SetAccountId(fmt.Sprintf("account_%d", i))
				account.SetName(fmt.Sprintf("Account %d", i))
				account.SetType(plaid.ACCOUNTTYPE_DEPOSITORY)

				balance := plaid.NewAccountBalanceWithDefaults()
				balance.SetIsoCurrencyCode("USD")
				account.SetBalances(*balance)

				sampleAccounts = append(sampleAccounts, *account)
			}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should return an error - list accounts error", func(ctx SpecContext) {
			fromPayload := models.OpenBankingForwardedUserFromPayload{
				OpenBankingConnection: &models.OpenBankingConnection{
					ConnectorID: models.ConnectorID{
						Reference: uuid.New(),
						Provider:  "plaid-test",
					},
					AccessToken:  &models.Token{Token: "test-token"},
					ConnectionID: "test-connection",
				},
				FromPayload: json.RawMessage(`{}`),
			}
			fromPayloadBytes, _ := json.Marshal(fromPayload)

			req := models.FetchNextAccountsRequest{
				FromPayload: fromPayloadBytes,
			}

			m.EXPECT().ListAccounts(gomock.Any(), "test-token").Return(
				plaid.AccountsGetResponse{},
				errors.New("test error"),
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextAccountsResponse{}))
		})

		It("should fetch accounts successfully", func(ctx SpecContext) {
			fromPayload := models.OpenBankingForwardedUserFromPayload{
				OpenBankingConnection: &models.OpenBankingConnection{
					ConnectorID: models.ConnectorID{
						Reference: uuid.New(),
						Provider:  "plaid-test",
					},
					AccessToken:  &models.Token{Token: "test-token"},
					ConnectionID: "test-connection",
				},
				FromPayload: json.RawMessage(`{}`),
			}
			fromPayloadBytes, _ := json.Marshal(fromPayload)

			req := models.FetchNextAccountsRequest{
				FromPayload: fromPayloadBytes,
			}

			m.EXPECT().ListAccounts(gomock.Any(), "test-token").Return(
				plaid.AccountsGetResponse{
					Accounts: sampleAccounts,
				},
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(3))
			Expect(resp.Accounts[0].Reference).To(Equal("account_0"))
			Expect(resp.Accounts[0].Name).ToNot(BeNil())
			Expect(*resp.Accounts[0].Name).To(Equal("Account 0"))
			Expect(resp.Accounts[0].Metadata["accountType"]).To(Equal("depository"))
			Expect(resp.Accounts[0].OpenBankingConnectionID).To(Not(BeNil()))
			Expect(*resp.Accounts[0].OpenBankingConnectionID).To(Equal("test-connection"))
			Expect(resp.Accounts[0].PsuID).To(Not(BeNil()))
			Expect(*resp.Accounts[0].PsuID).To(Equal(fromPayload.PSUID))
		})

		It("should handle empty accounts response", func(ctx SpecContext) {
			fromPayload := models.OpenBankingForwardedUserFromPayload{
				OpenBankingConnection: &models.OpenBankingConnection{
					ConnectorID: models.ConnectorID{
						Reference: uuid.New(),
						Provider:  "plaid-test",
					},
					AccessToken:  &models.Token{Token: "test-token"},
					ConnectionID: "test-connection",
				},
				FromPayload: json.RawMessage(`{}`),
			}
			fromPayloadBytes, _ := json.Marshal(fromPayload)

			req := models.FetchNextAccountsRequest{
				FromPayload: fromPayloadBytes,
			}

			m.EXPECT().ListAccounts(gomock.Any(), "test-token").Return(
				plaid.AccountsGetResponse{
					Accounts: []plaid.AccountBase{},
				},
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(0))
		})

		It("should handle invalid from payload", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				FromPayload: json.RawMessage(`invalid json`),
			}

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(resp).To(Equal(models.FetchNextAccountsResponse{}))
		})

		It("should handle invalid base webhook payload", func(ctx SpecContext) {
			fromPayload := models.OpenBankingForwardedUserFromPayload{
				OpenBankingConnection: &models.OpenBankingConnection{
					ConnectorID: models.ConnectorID{
						Reference: uuid.New(),
						Provider:  "plaid-test",
					},
					AccessToken:  &models.Token{Token: "test-token"},
					ConnectionID: "test-connection",
				},
				FromPayload: json.RawMessage(`invalid json`),
			}
			fromPayloadBytes, _ := json.Marshal(fromPayload)

			req := models.FetchNextAccountsRequest{
				FromPayload: fromPayloadBytes,
			}

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(resp).To(Equal(models.FetchNextAccountsResponse{}))
		})
	})
})
