package tink

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/payments/pkg/connectors/tink/client"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Tink *Plugin Payments", func() {
	Context("fetchNextPayments", func() {
		var (
			ctrl *gomock.Controller
			plg  connector.Plugin
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
			fromPayload := connector.OpenBankingForwardedUserFromPayload{
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

			req := connector.FetchNextPaymentsRequest{
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
			Expect(payment.Type).To(Equal(connector.PAYMENT_TYPE_PAYIN))
			Expect(payment.Status).To(Equal(connector.PAYMENT_STATUS_SUCCEEDED))
			Expect(payment.Scheme).To(Equal(connector.PAYMENT_SCHEME_OTHER))
			Expect(payment.Metadata).To(HaveLen(0)) // No ob provider psu metadata
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
			fromPayload := connector.OpenBankingForwardedUserFromPayload{
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

			req := connector.FetchNextPaymentsRequest{
				FromPayload: fromPayloadBytes,
				PageSize:    10,
				State:       nil,
			}

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))

			payment := resp.Payments[0]
			Expect(payment.Type).To(Equal(connector.PAYMENT_TYPE_PAYOUT))
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
			fromPayload := connector.OpenBankingForwardedUserFromPayload{
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

			req := connector.FetchNextPaymentsRequest{
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
			fromPayload := connector.OpenBankingForwardedUserFromPayload{
				FromPayload: webhookPayloadBytes,
			}
			fromPayloadBytes, err := json.Marshal(fromPayload)
			Expect(err).To(BeNil())

			// Mock the client error
			m.EXPECT().ListTransactions(gomock.Any(), gomock.Any()).Return(client.ListTransactionResponse{}, errors.New("client error"))

			req := connector.FetchNextPaymentsRequest{
				FromPayload: fromPayloadBytes,
				PageSize:    10,
				State:       nil,
			}

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(resp).To(Equal(connector.FetchNextPaymentsResponse{}))
		})

		It("should handle invalid from payload", func(ctx SpecContext) {
			req := connector.FetchNextPaymentsRequest{
				FromPayload: []byte("invalid json"),
				PageSize:    10,
				State:       nil,
			}

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(resp).To(Equal(connector.FetchNextPaymentsResponse{}))
		})

		It("should handle invalid webhook payload", func(ctx SpecContext) {
			// Create invalid from payload by directly using invalid JSON bytes
			fromPayloadBytes := []byte(`{"fromPayload": "invalid json"}`)

			req := connector.FetchNextPaymentsRequest{
				FromPayload: fromPayloadBytes,
				PageSize:    10,
				State:       nil,
			}

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(resp).To(Equal(connector.FetchNextPaymentsResponse{}))
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
			fromPayload := connector.OpenBankingForwardedUserFromPayload{
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

			req := connector.FetchNextPaymentsRequest{
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

			fromPayload := connector.OpenBankingForwardedUserFromPayload{
				PSUID: psuID,
				OpenBankingForwardedUser: &connector.OpenBankingForwardedUser{
					PsuID: psuID,
				},
				OpenBankingConnection: &connector.OpenBankingConnection{
					ConnectionID: connectionID,
				},
			}

			payments := make([]connector.PSPPayment, 0)
			result, err := toPSPPayments(payments, clientTransactions, fromPayload)

			Expect(err).To(BeNil())
			Expect(result).To(HaveLen(2))

			// Check first payment (positive amount - PAYIN)
			Expect(result[0].Reference).To(Equal("transaction1"))
			Expect(result[0].Type).To(Equal(connector.PAYMENT_TYPE_PAYIN))
			Expect(result[0].Status).To(Equal(connector.PAYMENT_STATUS_SUCCEEDED))
			Expect(result[0].Scheme).To(Equal(connector.PAYMENT_SCHEME_OTHER))
			Expect(*result[0].DestinationAccountReference).To(Equal(accountID))
			Expect(result[0].SourceAccountReference).To(BeNil())
			Expect(result[0].PsuID).To(Not(BeNil()))
			Expect(*result[0].PsuID).To(Equal(psuID))
			Expect(result[0].OpenBankingConnectionID).To(Not(BeNil()))
			Expect(*result[0].OpenBankingConnectionID).To(Equal(connectionID))
			Expect(result[0].Raw).ToNot(BeNil())

			// Check second payment (negative amount - PAYOUT)
			Expect(result[1].Reference).To(Equal("transaction2"))
			Expect(result[1].Type).To(Equal(connector.PAYMENT_TYPE_PAYOUT))
			Expect(result[1].Status).To(Equal(connector.PAYMENT_STATUS_PENDING))
			Expect(result[1].Scheme).To(Equal(connector.PAYMENT_SCHEME_OTHER))
			Expect(*result[1].SourceAccountReference).To(Equal(accountID))
			Expect(result[1].DestinationAccountReference).To(BeNil())
			Expect(result[1].PsuID).To(Not(BeNil()))
			Expect(*result[1].PsuID).To(Equal(psuID))
			Expect(result[1].OpenBankingConnectionID).To(Not(BeNil()))
			Expect(*result[1].OpenBankingConnectionID).To(Equal(connectionID))
			Expect(result[1].Raw).ToNot(BeNil())
		})

		It("should handle different transaction statuses", func(ctx SpecContext) {
			accountID := "test_account_id"

			testCases := []struct {
				status         string
				expectedStatus connector.PaymentStatus
				description    string
			}{
				{"BOOKED", connector.PAYMENT_STATUS_SUCCEEDED, "booked transaction"},
				{"PENDING", connector.PAYMENT_STATUS_PENDING, "pending transaction"},
				{"UNKNOWN", connector.PAYMENT_STATUS_OTHER, "unknown status"},
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

				fromPayload := connector.OpenBankingForwardedUserFromPayload{}

				payments := make([]connector.PSPPayment, 0)
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

			fromPayload := connector.OpenBankingForwardedUserFromPayload{}

			payments := make([]connector.PSPPayment, 0)
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

			fromPayload := connector.OpenBankingForwardedUserFromPayload{}

			payments := make([]connector.PSPPayment, 0)
			result, err := toPSPPayments(payments, []client.Transaction{clientTransaction}, fromPayload)

			Expect(err).ToNot(BeNil())
			Expect(result).To(HaveLen(0))
		})

		It("should handle missing open banking forwarded user metadata", func(ctx SpecContext) {
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

			fromPayload := connector.OpenBankingForwardedUserFromPayload{}

			payments := make([]connector.PSPPayment, 0)
			result, err := toPSPPayments(payments, []client.Transaction{clientTransaction}, fromPayload)

			Expect(err).To(BeNil())
			Expect(result).To(HaveLen(1))

			payment := result[0]
			Expect(payment.Metadata).To(HaveLen(0))
		})
	})

	Context("computeCreatedAt", func() {
		It("should return TransactionDateTime when it is set", func() {
			expectedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
			transaction := client.Transaction{
				ID:                  "test_id",
				AccountID:           "test_account",
				Status:              "BOOKED",
				TransactionDateTime: expectedTime,
				BookedDateTime:      time.Date(2024, 1, 14, 9, 0, 0, 0, time.UTC),
				Dates: client.TransactionDates{
					Transaction: time.Date(2024, 1, 13, 0, 0, 0, 0, time.UTC),
					Booked:      time.Date(2024, 1, 12, 0, 0, 0, 0, time.UTC),
				},
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
			}

			result := computeCreatedAt(transaction)
			Expect(result).To(Equal(expectedTime))
		})

		It("should return Dates.Transaction when TransactionDateTime is zero", func() {
			expectedTime := time.Date(2024, 1, 13, 0, 0, 0, 0, time.UTC)
			transaction := client.Transaction{
				ID:                  "test_id",
				AccountID:           "test_account",
				Status:              "BOOKED",
				TransactionDateTime: time.Time{}, // Zero time
				BookedDateTime:      time.Date(2024, 1, 14, 9, 0, 0, 0, time.UTC),
				Dates: client.TransactionDates{
					Transaction: expectedTime,
					Booked:      time.Date(2024, 1, 12, 0, 0, 0, 0, time.UTC),
				},
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
			}

			result := computeCreatedAt(transaction)
			Expect(result).To(Equal(expectedTime))
		})

		It("should return BookedDateTime when TransactionDateTime and Dates.Transaction are zero", func() {
			expectedTime := time.Date(2024, 1, 14, 9, 0, 0, 0, time.UTC)
			transaction := client.Transaction{
				ID:                  "test_id",
				AccountID:           "test_account",
				Status:              "BOOKED",
				TransactionDateTime: time.Time{}, // Zero time
				BookedDateTime:      expectedTime,
				Dates: client.TransactionDates{
					Transaction: time.Time{}, // Zero time
					Booked:      time.Date(2024, 1, 12, 0, 0, 0, 0, time.UTC),
				},
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
			}

			result := computeCreatedAt(transaction)
			Expect(result).To(Equal(expectedTime))
		})

		It("should return Dates.Booked when all DateTime fields are zero", func() {
			expectedTime := time.Date(2024, 1, 12, 0, 0, 0, 0, time.UTC)
			transaction := client.Transaction{
				ID:                  "test_id",
				AccountID:           "test_account",
				Status:              "BOOKED",
				TransactionDateTime: time.Time{}, // Zero time
				BookedDateTime:      time.Time{}, // Zero time
				Dates: client.TransactionDates{
					Transaction: time.Time{}, // Zero time
					Booked:      expectedTime,
				},
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
			}

			result := computeCreatedAt(transaction)
			Expect(result).To(Equal(expectedTime))
		})

		It("should return time.Now() when all date fields are zero", func() {
			before := time.Now()
			transaction := client.Transaction{
				ID:                  "test_id",
				AccountID:           "test_account",
				Status:              "BOOKED",
				TransactionDateTime: time.Time{}, // Zero time
				BookedDateTime:      time.Time{}, // Zero time
				Dates: client.TransactionDates{
					Transaction: time.Time{}, // Zero time
					Booked:      time.Time{}, // Zero time
				},
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
			}

			result := computeCreatedAt(transaction)
			after := time.Now()

			Expect(result).To(BeTemporally(">=", before))
			Expect(result).To(BeTemporally("<=", after))
		})
	})
})
