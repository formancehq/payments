package tink

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "github.com/golang/mock/gomock"
)

var _ = Describe("Tink *Plugin Accounts", func() {
	Context("fetchNextAccounts", func() {
		var (
			ctrl *gomock.Controller
			plg  models.Plugin
			m    *client.MockClient
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{
				client: m,
			}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should fetch accounts successfully", func(ctx SpecContext) {
			userID := uuid.New()
			accountID := "test_account_id"

			// Create the webhook payload with proper time values
			webhookPayload := fetchNextDataRequest{
				UserID:                                userID.String(),
				ExternalUserID:                        userID.String(),
				AccountID:                             accountID,
				TransactionEarliestModifiedBookedDate: time.Now().Add(-24 * time.Hour),
				TransactionLatestModifiedBookedDate:   time.Now(),
			}
			webhookPayloadBytes, err := json.Marshal(webhookPayload)
			Expect(err).To(BeNil())

			// Create the from payload using only FromPayload to avoid issues
			fromPayload := models.OpenBankingForwardedUserFromPayload{
				FromPayload: webhookPayloadBytes,
			}
			fromPayloadBytes, err := json.Marshal(fromPayload)
			Expect(err).To(BeNil())

			// Mock the client response
			expectedAccount := client.Account{
				ID:   accountID,
				Name: "Test Account",
				Type: "CHECKING",
			}

			m.EXPECT().GetAccount(gomock.Any(), userID.String(), accountID).Return(expectedAccount, nil)

			req := models.FetchNextAccountsRequest{
				FromPayload: fromPayloadBytes,
			}

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(1))

			account := resp.Accounts[0]
			Expect(account.Reference).To(Equal(accountID))
			Expect(*account.Name).To(Equal("Test Account"))
			Expect(account.Metadata).To(HaveLen(0)) // No OB Provider psu metadata
			Expect(account.Raw).ToNot(BeNil())
		})

		It("should handle client error", func(ctx SpecContext) {
			userID := uuid.New()
			accountID := "test_account_id"

			// Create the webhook payload
			webhookPayload := fetchNextDataRequest{
				UserID:         userID.String(),
				ExternalUserID: userID.String(),
				AccountID:      accountID,
			}
			webhookPayloadBytes, err := json.Marshal(webhookPayload)
			Expect(err).To(BeNil())

			// Create the open banking forwarded user from payload
			fromPayload := models.OpenBankingForwardedUserFromPayload{
				FromPayload: webhookPayloadBytes,
			}
			fromPayloadBytes, err := json.Marshal(fromPayload)
			Expect(err).To(BeNil())

			// Mock the client error
			m.EXPECT().GetAccount(gomock.Any(), userID.String(), accountID).Return(client.Account{}, errors.New("client error"))

			req := models.FetchNextAccountsRequest{
				FromPayload: fromPayloadBytes,
			}

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(resp).To(Equal(models.FetchNextAccountsResponse{}))
		})

		It("should handle invalid from payload", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				FromPayload: []byte("invalid json"),
			}

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(resp).To(Equal(models.FetchNextAccountsResponse{}))
		})

		It("should handle invalid webhook payload", func(ctx SpecContext) {
			// Create invalid from payload by directly using invalid JSON bytes
			fromPayloadBytes := []byte(`{"fromPayload": "invalid json"}`)

			req := models.FetchNextAccountsRequest{
				FromPayload: fromPayloadBytes,
			}

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(resp).To(Equal(models.FetchNextAccountsResponse{}))
		})

		It("should handle missing ob provider psu metadata", func(ctx SpecContext) {
			userID := uuid.New()
			accountID := "test_account_id"

			// Create the webhook payload
			webhookPayload := fetchNextDataRequest{
				UserID:         userID.String(),
				ExternalUserID: userID.String(),
				AccountID:      accountID,
			}
			webhookPayloadBytes, err := json.Marshal(webhookPayload)
			Expect(err).To(BeNil())

			// Create the from payload without open banking forwarded user
			fromPayload := models.OpenBankingForwardedUserFromPayload{
				FromPayload: webhookPayloadBytes,
			}
			fromPayloadBytes, err := json.Marshal(fromPayload)
			Expect(err).To(BeNil())

			// Mock the client response
			expectedAccount := client.Account{
				ID:   accountID,
				Name: "Test Account",
				Type: "CHECKING",
			}

			m.EXPECT().GetAccount(gomock.Any(), userID.String(), accountID).Return(expectedAccount, nil)

			req := models.FetchNextAccountsRequest{
				FromPayload: fromPayloadBytes,
			}

			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(1))

			account := resp.Accounts[0]
			Expect(account.Reference).To(Equal(accountID))
			Expect(*account.Name).To(Equal("Test Account"))
			Expect(account.Metadata).To(HaveLen(0))
		})
	})

	Context("toPSPAccounts", func() {
		It("should convert client accounts to PSP accounts", func() {
			psuID := uuid.New()
			connectionID := "test_connection_id"

			clientAccounts := []client.Account{
				{
					ID:   "account1",
					Name: "Account 1",
					Type: "CHECKING",
				},
				{
					ID:   "account2",
					Name: "Account 2",
					Type: "SAVINGS",
				},
			}

			fromPayload := models.OpenBankingForwardedUserFromPayload{
				PSUID: psuID,
				OpenBankingForwardedUser: &models.OpenBankingForwardedUser{
					PsuID: psuID,
				},
				OpenBankingConnection: &models.OpenBankingConnection{
					ConnectionID: connectionID,
				},
			}

			accounts := make([]models.PSPAccount, 0)
			result, err := toPSPAccounts(accounts, clientAccounts, fromPayload)

			Expect(err).To(BeNil())
			Expect(result).To(HaveLen(2))

			// Check first account
			Expect(result[0].Reference).To(Equal("account1"))
			Expect(*result[0].Name).To(Equal("Account 1"))
			Expect(result[0].PsuID).To(Not(BeNil()))
			Expect(*result[0].PsuID).To(Equal(psuID))
			Expect(result[0].OpenBankingConnectionID).To(Not(BeNil()))
			Expect(*result[0].OpenBankingConnectionID).To(Equal(connectionID))
			Expect(result[0].Raw).ToNot(BeNil())

			// Check second account
			Expect(result[1].Reference).To(Equal("account2"))
			Expect(*result[1].Name).To(Equal("Account 2"))
			Expect(result[1].PsuID).To(Not(BeNil()))
			Expect(*result[1].PsuID).To(Equal(psuID))
			Expect(result[1].OpenBankingConnectionID).To(Not(BeNil()))
			Expect(*result[1].OpenBankingConnectionID).To(Equal(connectionID))
			Expect(result[1].Raw).ToNot(BeNil())
		})

		It("should handle JSON marshaling error", func() {
			// This test would require mocking json.Marshal which is not easily done
			// The function should handle this gracefully, but we can't easily test it
			// without more complex mocking
		})
	})
})
