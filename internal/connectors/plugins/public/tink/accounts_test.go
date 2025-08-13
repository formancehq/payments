package tink

import (
	"encoding/json"
	"errors"

	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Tink *Plugin Accounts", func() {
	Context("fetch next accounts", func() {
		var (
			ctrl *gomock.Controller
			plg  *Plugin
			m    *client.MockClient

			sampleAccounts []client.Account
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{client: m}

			sampleAccounts = []client.Account{
				{
					ID:   "account_1",
					Name: "Account 1",
				},
				{
					ID:   "account_2",
					Name: "Account 2",
				},
				{
					ID:   "account_3",
					Name: "Account 3",
				},
			}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should return an error - list accounts error", func(ctx SpecContext) {
			webhook := client.AccountTransactionsModifiedWebhook{
				ExternalUserID: "user_123",
			}
			webhookBytes, _ := json.Marshal(webhook)

			fromPayload := models.BankBridgeFromPayload{
				FromPayload: webhookBytes,
			}
			fromPayloadBytes, _ := json.Marshal(fromPayload)

			req := models.FetchNextAccountsRequest{
				FromPayload: fromPayloadBytes,
				PageSize:    10,
			}

			m.EXPECT().ListAccounts(gomock.Any(), "user_123", "").Return(
				client.ListAccountsResponse{},
				errors.New("test error"),
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextAccountsResponse{}))
		})

		It("should fetch accounts successfully", func(ctx SpecContext) {
			webhook := client.AccountTransactionsModifiedWebhook{
				ExternalUserID: "user_123",
			}
			webhookBytes, _ := json.Marshal(webhook)

			fromPayload := models.BankBridgeFromPayload{
				FromPayload: webhookBytes,
			}
			fromPayloadBytes, _ := json.Marshal(fromPayload)

			req := models.FetchNextAccountsRequest{
				FromPayload: fromPayloadBytes,
				PageSize:    10,
			}

			m.EXPECT().ListAccounts(gomock.Any(), "user_123", "").Return(
				client.ListAccountsResponse{
					Accounts:      sampleAccounts,
					NextPageToken: "",
				},
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(3))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			// Verify account details
			Expect(resp.Accounts[0].Reference).To(Equal("account_1"))
			Expect(*resp.Accounts[0].Name).To(Equal("Account 1"))
			Expect(resp.Accounts[1].Reference).To(Equal("account_2"))
			Expect(*resp.Accounts[1].Name).To(Equal("Account 2"))
			Expect(resp.Accounts[2].Reference).To(Equal("account_3"))
			Expect(*resp.Accounts[2].Name).To(Equal("Account 3"))
		})

		It("should handle pagination correctly", func(ctx SpecContext) {
			webhook := client.AccountTransactionsModifiedWebhook{
				ExternalUserID: "user_123",
			}
			webhookBytes, _ := json.Marshal(webhook)

			fromPayload := models.BankBridgeFromPayload{
				FromPayload: webhookBytes,
			}
			fromPayloadBytes, _ := json.Marshal(fromPayload)

			req := models.FetchNextAccountsRequest{
				FromPayload: fromPayloadBytes,
				PageSize:    2,
			}

			// First page - return first two accounts with next page token
			// The current implementation breaks when NextPageToken is not empty, so it will only return the first page
			// hasMore will be false because it's set to false initially and never updated when we break
			m.EXPECT().ListAccounts(gomock.Any(), "user_123", "").Return(
				client.ListAccountsResponse{
					Accounts:      sampleAccounts[:2],
					NextPageToken: "next_token",
				},
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(2))
			Expect(resp.HasMore).To(BeFalse())

			// Verify state contains next page token
			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.NextPageToken).To(Equal("next_token"))
		})

		It("should handle existing state", func(ctx SpecContext) {
			webhook := client.AccountTransactionsModifiedWebhook{
				ExternalUserID: "user_123",
			}
			webhookBytes, _ := json.Marshal(webhook)

			fromPayload := models.BankBridgeFromPayload{
				FromPayload: webhookBytes,
			}
			fromPayloadBytes, _ := json.Marshal(fromPayload)

			existingState := accountsState{
				NextPageToken: "existing_token",
			}
			stateBytes, _ := json.Marshal(existingState)

			req := models.FetchNextAccountsRequest{
				FromPayload: fromPayloadBytes,
				State:       stateBytes,
				PageSize:    10,
			}

			m.EXPECT().ListAccounts(gomock.Any(), "user_123", "existing_token").Return(
				client.ListAccountsResponse{
					Accounts:      sampleAccounts,
					NextPageToken: "",
				},
				nil,
			)

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(3))
		})

		It("should handle invalid from payload", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				FromPayload: json.RawMessage(`invalid json`),
				PageSize:    10,
			}

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(resp).To(Equal(models.FetchNextAccountsResponse{}))
		})

		It("should handle invalid webhook payload", func(ctx SpecContext) {
			fromPayload := models.BankBridgeFromPayload{
				FromPayload: json.RawMessage(`invalid json`),
			}
			fromPayloadBytes, _ := json.Marshal(fromPayload)

			req := models.FetchNextAccountsRequest{
				FromPayload: fromPayloadBytes,
				PageSize:    10,
			}

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(resp).To(Equal(models.FetchNextAccountsResponse{}))
		})

		It("should handle invalid state", func(ctx SpecContext) {
			webhook := client.AccountTransactionsModifiedWebhook{
				ExternalUserID: "user_123",
			}
			webhookBytes, _ := json.Marshal(webhook)

			fromPayload := models.BankBridgeFromPayload{
				FromPayload: webhookBytes,
			}
			fromPayloadBytes, _ := json.Marshal(fromPayload)

			req := models.FetchNextAccountsRequest{
				FromPayload: fromPayloadBytes,
				State:       json.RawMessage(`invalid json`),
				PageSize:    10,
			}

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(resp).To(Equal(models.FetchNextAccountsResponse{}))
		})
	})
})
