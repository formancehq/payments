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
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Tink *Plugin Payments", func() {
	Context("fetchNextPayments", func() {
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

		It("should fetch payments successfully", func(ctx SpecContext) {
			userID := uuid.New()
			accountID := "test_account_id"
			transactionID := "test_transaction_id"
			earliestDate := time.Now().Add(-24 * time.Hour)
			latestDate := time.Now()

			// Create the webhook payload
			webhookPayload := fetchNextDataRequest{
				UserID:                                userID.String(),
				ExternalUserID:                        userID.String(),
				AccountID:                             accountID,
				TransactionEarliestModifiedBookedDate: earliestDate,
				TransactionLatestModifiedBookedDate:   latestDate,
			}
			webhookPayloadBytes, err := json.Marshal(webhookPayload)
			Expect(err).To(BeNil())

			// Create the from payload using only FromPayload to avoid issues
			fromPayload := models.BankBridgeFromPayload{
				FromPayload: webhookPayloadBytes,
			}
			fromPayloadBytes, err := json.Marshal(fromPayload)
			Expect(err).To(BeNil())

			// Mock the client response
			expectedTransaction := client.Transaction{
				ID:                  transactionID,
				AccountID:           accountID,
				Status:              "BOOKED",
				BookedDateTime:      time.Now(),
				TransactionDateTime: time.Now(),
				ValueDateTime:       time.Now(),
				Amount: client.Amount{
					CurrencyCode: "EUR",
					Value: struct {
						Scale string `json:"scale"`
						Value string `json:"unscaledValue"`
					}{
						Scale: "2",
						Value: "1000",
					},
				},
				Descriptions: client.Descriptions{
					Detailed: struct {
						Unstructured string `json:"unstructured"`
					}{
						Unstructured: "Test transaction",
					},
					Display:  "Test transaction",
					Original: "Test transaction",
				},
			}

			expectedResponse := client.ListTransactionResponse{
				NextPageToken: "",
				Transactions:  []client.Transaction{expectedTransaction},
			}

			m.EXPECT().ListTransactions(gomock.Any(), gomock.Any()).Return(expectedResponse, nil)

			req := models.FetchNextPaymentsRequest{
				FromPayload: fromPayloadBytes,
				PageSize:    10,
				State:       nil,
			}

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			Expect(resp.HasMore).To(BeFalse())

			payment := resp.Payments[0]
			Expect(payment.Reference).To(Equal(transactionID))
			Expect(payment.Type).To(Equal(models.PAYMENT_TYPE_PAYIN))
			Expect(payment.Status).To(Equal(models.PAYMENT_STATUS_SUCCEEDED))
			Expect(payment.Scheme).To(Equal(models.PAYMENT_SCHEME_OTHER))
			Expect(payment.Metadata).To(HaveLen(0)) // No PSU bank bridge metadata
			Expect(payment.Raw).ToNot(BeNil())
		})

		It("should handle payout transactions (negative amount)", func(ctx SpecContext) {
			userID := uuid.New()
			accountID := "test_account_id"
			transactionID := "test_transaction_id"
			earliestDate := time.Now().Add(-24 * time.Hour)
			latestDate := time.Now()

			// Create the webhook payload
			webhookPayload := fetchNextDataRequest{
				UserID:                                userID.String(),
				ExternalUserID:                        userID.String(),
				AccountID:                             accountID,
				TransactionEarliestModifiedBookedDate: earliestDate,
				TransactionLatestModifiedBookedDate:   latestDate,
			}
			webhookPayloadBytes, err := json.Marshal(webhookPayload)
			Expect(err).To(BeNil())

			// Create the from payload
			fromPayload := models.BankBridgeFromPayload{
				FromPayload: webhookPayloadBytes,
			}
			fromPayloadBytes, err := json.Marshal(fromPayload)
			Expect(err).To(BeNil())

			// Mock the client response with negative amount
			expectedTransaction := client.Transaction{
				ID:                  transactionID,
				AccountID:           accountID,
				Status:              "BOOKED",
				BookedDateTime:      time.Now(),
				TransactionDateTime: time.Now(),
				ValueDateTime:       time.Now(),
				Amount: client.Amount{
					CurrencyCode: "EUR",
					Value: struct {
						Scale string `json:"scale"`
						Value string `json:"unscaledValue"`
					}{
						Scale: "2",
						Value: "-1000", // Negative amount
					},
				},
				Descriptions: client.Descriptions{
					Detailed: struct {
						Unstructured string `json:"unstructured"`
					}{
						Unstructured: "Test payout",
					},
					Display:  "Test payout",
					Original: "Test payout",
				},
			}

			expectedResponse := client.ListTransactionResponse{
				NextPageToken: "",
				Transactions:  []client.Transaction{expectedTransaction},
			}

			m.EXPECT().ListTransactions(gomock.Any(), gomock.Any()).Return(expectedResponse, nil)

			req := models.FetchNextPaymentsRequest{
				FromPayload: fromPayloadBytes,
				PageSize:    10,
				State:       nil,
			}

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))

			payment := resp.Payments[0]
			Expect(payment.Type).To(Equal(models.PAYMENT_TYPE_PAYOUT))
			Expect(*payment.SourceAccountReference).To(Equal(accountID))
			Expect(payment.DestinationAccountReference).To(BeNil())
		})

		It("should handle pagination", func(ctx SpecContext) {
			userID := uuid.New()
			accountID := "test_account_id"
			earliestDate := time.Now().Add(-24 * time.Hour)
			latestDate := time.Now()

			// Create the webhook payload
			webhookPayload := fetchNextDataRequest{
				UserID:                                userID.String(),
				ExternalUserID:                        userID.String(),
				AccountID:                             accountID,
				TransactionEarliestModifiedBookedDate: earliestDate,
				TransactionLatestModifiedBookedDate:   latestDate,
			}
			webhookPayloadBytes, err := json.Marshal(webhookPayload)
			Expect(err).To(BeNil())

			// Create the from payload
			fromPayload := models.BankBridgeFromPayload{
				FromPayload: webhookPayloadBytes,
			}
			fromPayloadBytes, err := json.Marshal(fromPayload)
			Expect(err).To(BeNil())

			// Mock first page response
			firstPageResponse := client.ListTransactionResponse{
				NextPageToken: "next_page_token",
				Transactions:  []client.Transaction{},
			}

			// Mock second page response
			secondPageResponse := client.ListTransactionResponse{
				NextPageToken: "",
				Transactions:  []client.Transaction{},
			}

			m.EXPECT().ListTransactions(gomock.Any(), gomock.Any()).Return(firstPageResponse, nil)
			m.EXPECT().ListTransactions(gomock.Any(), gomock.Any()).Return(secondPageResponse, nil)

			req := models.FetchNextPaymentsRequest{
				FromPayload: fromPayloadBytes,
				PageSize:    10,
				State:       nil,
			}

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.HasMore).To(BeFalse())
		})

		It("should handle client error", func(ctx SpecContext) {
			userID := uuid.New()
			accountID := "test_account_id"
			earliestDate := time.Now().Add(-24 * time.Hour)
			latestDate := time.Now()

			// Create the webhook payload
			webhookPayload := fetchNextDataRequest{
				UserID:                                userID.String(),
				ExternalUserID:                        userID.String(),
				AccountID:                             accountID,
				TransactionEarliestModifiedBookedDate: earliestDate,
				TransactionLatestModifiedBookedDate:   latestDate,
			}
			webhookPayloadBytes, err := json.Marshal(webhookPayload)
			Expect(err).To(BeNil())

			// Create the from payload
			fromPayload := models.BankBridgeFromPayload{
				FromPayload: webhookPayloadBytes,
			}
			fromPayloadBytes, err := json.Marshal(fromPayload)
			Expect(err).To(BeNil())

			// Mock the client error
			m.EXPECT().ListTransactions(gomock.Any(), gomock.Any()).Return(client.ListTransactionResponse{}, errors.New("client error"))

			req := models.FetchNextPaymentsRequest{
				FromPayload: fromPayloadBytes,
				PageSize:    10,
				State:       nil,
			}

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should handle invalid from payload", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				FromPayload: []byte("invalid json"),
				PageSize:    10,
				State:       nil,
			}

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should handle invalid webhook payload", func(ctx SpecContext) {
			// Create invalid from payload by directly using invalid JSON bytes
			fromPayloadBytes := []byte(`{"fromPayload": "invalid json"}`)

			req := models.FetchNextPaymentsRequest{
				FromPayload: fromPayloadBytes,
				PageSize:    10,
				State:       nil,
			}

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should handle state with next page token", func(ctx SpecContext) {
			userID := uuid.New()
			accountID := "test_account_id"
			earliestDate := time.Now().Add(-24 * time.Hour)
			latestDate := time.Now()
			nextPageToken := "existing_page_token"

			// Create the webhook payload
			webhookPayload := fetchNextDataRequest{
				UserID:                                userID.String(),
				ExternalUserID:                        userID.String(),
				AccountID:                             accountID,
				TransactionEarliestModifiedBookedDate: earliestDate,
				TransactionLatestModifiedBookedDate:   latestDate,
			}
			webhookPayloadBytes, err := json.Marshal(webhookPayload)
			Expect(err).To(BeNil())

			// Create the from payload
			fromPayload := models.BankBridgeFromPayload{
				FromPayload: webhookPayloadBytes,
			}
			fromPayloadBytes, err := json.Marshal(fromPayload)
			Expect(err).To(BeNil())

			// Create state with next page token
			state := paymentsState{
				NextPageToken: nextPageToken,
			}
			stateBytes, err := json.Marshal(state)
			Expect(err).To(BeNil())

			// Mock the client response
			expectedResponse := client.ListTransactionResponse{
				NextPageToken: "",
				Transactions:  []client.Transaction{},
			}

			m.EXPECT().ListTransactions(gomock.Any(), gomock.Any()).Return(expectedResponse, nil)

			req := models.FetchNextPaymentsRequest{
				FromPayload: fromPayloadBytes,
				PageSize:    10,
				State:       stateBytes,
			}

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.HasMore).To(BeFalse())
		})
	})

	Context("toPSPPayments", func() {
		It("should convert client transactions to PSP payments", func() {
			psuID := uuid.New()
			connectionID := "test_connection_id"
			accountID := "test_account_id"

			clientTransactions := []client.Transaction{
				{
					ID:                  "transaction1",
					AccountID:           accountID,
					Status:              "BOOKED",
					BookedDateTime:      time.Now(),
					TransactionDateTime: time.Now(),
					ValueDateTime:       time.Now(),
					Amount: client.Amount{
						CurrencyCode: "EUR",
						Value: struct {
							Scale string `json:"scale"`
							Value string `json:"unscaledValue"`
						}{
							Scale: "2",
							Value: "1000",
						},
					},
					Descriptions: client.Descriptions{
						Detailed: struct {
							Unstructured string `json:"unstructured"`
						}{
							Unstructured: "Test transaction 1",
						},
						Display:  "Test transaction 1",
						Original: "Test transaction 1",
					},
				},
				{
					ID:                  "transaction2",
					AccountID:           accountID,
					Status:              "PENDING",
					BookedDateTime:      time.Now(),
					TransactionDateTime: time.Now(),
					ValueDateTime:       time.Now(),
					Amount: client.Amount{
						CurrencyCode: "USD",
						Value: struct {
							Scale string `json:"scale"`
							Value string `json:"unscaledValue"`
						}{
							Scale: "2",
							Value: "-500", // Negative amount
						},
					},
					Descriptions: client.Descriptions{
						Detailed: struct {
							Unstructured string `json:"unstructured"`
						}{
							Unstructured: "Test transaction 2",
						},
						Display:  "Test transaction 2",
						Original: "Test transaction 2",
					},
				},
			}

			fromPayload := models.BankBridgeFromPayload{
				PSUBankBridge: &models.PSUBankBridge{
					PsuID: psuID,
				},
				PSUBankBridgeConnection: &models.PSUBankBridgeConnection{
					ConnectionID: connectionID,
				},
			}

			payments := make([]models.PSPPayment, 0)
			result, err := toPSPPayments(payments, clientTransactions, fromPayload)

			Expect(err).To(BeNil())
			Expect(result).To(HaveLen(2))

			// Check first payment (positive amount - PAYIN)
			Expect(result[0].Reference).To(Equal("transaction1"))
			Expect(result[0].Type).To(Equal(models.PAYMENT_TYPE_PAYIN))
			Expect(result[0].Status).To(Equal(models.PAYMENT_STATUS_SUCCEEDED))
			Expect(result[0].Scheme).To(Equal(models.PAYMENT_SCHEME_OTHER))
			Expect(*result[0].DestinationAccountReference).To(Equal(accountID))
			Expect(result[0].SourceAccountReference).To(BeNil())
			Expect(result[0].Metadata[models.ObjectPSUIDMetadataKey]).To(Equal(psuID.String()))
			Expect(result[0].Metadata[models.ObjectConnectionIDMetadataKey]).To(Equal(connectionID))
			Expect(result[0].Raw).ToNot(BeNil())

			// Check second payment (negative amount - PAYOUT)
			Expect(result[1].Reference).To(Equal("transaction2"))
			Expect(result[1].Type).To(Equal(models.PAYMENT_TYPE_PAYOUT))
			Expect(result[1].Status).To(Equal(models.PAYMENT_STATUS_PENDING))
			Expect(result[1].Scheme).To(Equal(models.PAYMENT_SCHEME_OTHER))
			Expect(*result[1].SourceAccountReference).To(Equal(accountID))
			Expect(result[1].DestinationAccountReference).To(BeNil())
			Expect(result[1].Metadata[models.ObjectPSUIDMetadataKey]).To(Equal(psuID.String()))
			Expect(result[1].Metadata[models.ObjectConnectionIDMetadataKey]).To(Equal(connectionID))
			Expect(result[1].Raw).ToNot(BeNil())
		})

		It("should handle different transaction statuses", func(ctx SpecContext) {
			accountID := "test_account_id"

			testCases := []struct {
				status         string
				expectedStatus models.PaymentStatus
				description    string
			}{
				{"BOOKED", models.PAYMENT_STATUS_SUCCEEDED, "booked transaction"},
				{"PENDING", models.PAYMENT_STATUS_PENDING, "pending transaction"},
				{"UNKNOWN", models.PAYMENT_STATUS_OTHER, "unknown status"},
			}

			for _, tc := range testCases {
				clientTransaction := client.Transaction{
					ID:                  "transaction_" + tc.status,
					AccountID:           accountID,
					Status:              tc.status,
					BookedDateTime:      time.Now(),
					TransactionDateTime: time.Now(),
					ValueDateTime:       time.Now(),
					Amount: client.Amount{
						CurrencyCode: "EUR",
						Value: struct {
							Scale string `json:"scale"`
							Value string `json:"unscaledValue"`
						}{
							Scale: "2",
							Value: "1000",
						},
					},
					Descriptions: client.Descriptions{
						Detailed: struct {
							Unstructured string `json:"unstructured"`
						}{
							Unstructured: tc.description,
						},
						Display:  tc.description,
						Original: tc.description,
					},
				}

				fromPayload := models.BankBridgeFromPayload{}

				payments := make([]models.PSPPayment, 0)
				result, err := toPSPPayments(payments, []client.Transaction{clientTransaction}, fromPayload)

				Expect(err).To(BeNil())
				Expect(result).To(HaveLen(1))
				Expect(result[0].Status).To(Equal(tc.expectedStatus), "for status: %s", tc.status)
			}
		})

		It("should handle invalid amount scale", func(ctx SpecContext) {
			accountID := "test_account_id"

			clientTransaction := client.Transaction{
				ID:                  "transaction_invalid_scale",
				AccountID:           accountID,
				Status:              "BOOKED",
				BookedDateTime:      time.Now(),
				TransactionDateTime: time.Now(),
				ValueDateTime:       time.Now(),
				Amount: client.Amount{
					CurrencyCode: "EUR",
					Value: struct {
						Scale string `json:"scale"`
						Value string `json:"unscaledValue"`
					}{
						Scale: "invalid_scale", // Invalid scale
						Value: "1000",
					},
				},
				Descriptions: client.Descriptions{
					Detailed: struct {
						Unstructured string `json:"unstructured"`
					}{
						Unstructured: "Test transaction",
					},
					Display:  "Test transaction",
					Original: "Test transaction",
				},
			}

			fromPayload := models.BankBridgeFromPayload{}

			payments := make([]models.PSPPayment, 0)
			result, err := toPSPPayments(payments, []client.Transaction{clientTransaction}, fromPayload)

			Expect(err).ToNot(BeNil())
			Expect(result).To(HaveLen(0))
		})

		It("should handle invalid amount value", func(ctx SpecContext) {
			accountID := "test_account_id"

			clientTransaction := client.Transaction{
				ID:                  "transaction_invalid_value",
				AccountID:           accountID,
				Status:              "BOOKED",
				BookedDateTime:      time.Now(),
				TransactionDateTime: time.Now(),
				ValueDateTime:       time.Now(),
				Amount: client.Amount{
					CurrencyCode: "EUR",
					Value: struct {
						Scale string `json:"scale"`
						Value string `json:"unscaledValue"`
					}{
						Scale: "2",
						Value: "invalid_value", // Invalid value
					},
				},
				Descriptions: client.Descriptions{
					Detailed: struct {
						Unstructured string `json:"unstructured"`
					}{
						Unstructured: "Test transaction",
					},
					Display:  "Test transaction",
					Original: "Test transaction",
				},
			}

			fromPayload := models.BankBridgeFromPayload{}

			payments := make([]models.PSPPayment, 0)
			result, err := toPSPPayments(payments, []client.Transaction{clientTransaction}, fromPayload)

			Expect(err).ToNot(BeNil())
			Expect(result).To(HaveLen(0))
		})

		It("should handle missing PSU bank bridge metadata", func(ctx SpecContext) {
			accountID := "test_account_id"

			clientTransaction := client.Transaction{
				ID:                  "transaction_no_metadata",
				AccountID:           accountID,
				Status:              "BOOKED",
				BookedDateTime:      time.Now(),
				TransactionDateTime: time.Now(),
				ValueDateTime:       time.Now(),
				Amount: client.Amount{
					CurrencyCode: "EUR",
					Value: struct {
						Scale string `json:"scale"`
						Value string `json:"unscaledValue"`
					}{
						Scale: "2",
						Value: "1000",
					},
				},
				Descriptions: client.Descriptions{
					Detailed: struct {
						Unstructured string `json:"unstructured"`
					}{
						Unstructured: "Test transaction",
					},
					Display:  "Test transaction",
					Original: "Test transaction",
				},
			}

			fromPayload := models.BankBridgeFromPayload{}

			payments := make([]models.PSPPayment, 0)
			result, err := toPSPPayments(payments, []client.Transaction{clientTransaction}, fromPayload)

			Expect(err).To(BeNil())
			Expect(result).To(HaveLen(1))

			payment := result[0]
			Expect(payment.Metadata).To(HaveLen(0))
		})
	})
})
