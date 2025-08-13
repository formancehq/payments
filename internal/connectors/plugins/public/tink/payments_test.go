package tink

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Tink *Plugin Payments", func() {
	Context("fetch next payments", func() {
		var (
			ctrl *gomock.Controller
			plg  *Plugin
			m    *client.MockClient

			sampleTransactions []client.Transaction
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{client: m}

			sampleTransactions = []client.Transaction{
				{
					ID: "transaction_1",
					Amount: client.Amount{
						CurrencyCode: "EUR",
						Value: struct {
							Scale string `json:"scale"`
							Value string `json:"unscaledValue"`
						}{
							Value: "1000",
							Scale: "2",
						},
					},
					Status:              "BOOKED",
					AccountID:           "account_1",
					TransactionDateTime: time.Now().UTC(),
					BookedDateTime:      time.Now().UTC(),
					ValueDateTime:       time.Now().UTC(),
					Descriptions: client.Descriptions{
						Display: "Payment 1",
					},
				},
				{
					ID: "transaction_2",
					Amount: client.Amount{
						CurrencyCode: "EUR",
						Value: struct {
							Scale string `json:"scale"`
							Value string `json:"unscaledValue"`
						}{
							Value: "-500",
							Scale: "2",
						},
					},
					Status:              "PENDING",
					AccountID:           "account_2",
					TransactionDateTime: time.Now().UTC(),
					BookedDateTime:      time.Now().UTC(),
					ValueDateTime:       time.Now().UTC(),
					Descriptions: client.Descriptions{
						Display: "Payment 2",
					},
				},
			}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should return an error - list transactions error", func(ctx SpecContext) {
			webhook := client.AccountTransactionsModifiedWebhook{
				ExternalUserID: "user_123",
				Account: struct {
					ID string `json:"id"`
				}{
					ID: "account_1",
				},
				Transactions: client.WebhookTransactions{
					EarliestModifiedBookedDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					LatestModifiedBookedDate:   time.Date(2023, 1, 31, 0, 0, 0, 0, time.UTC),
				},
			}
			webhookBytes, _ := json.Marshal(webhook)

			fromPayload := models.BankBridgeFromPayload{
				FromPayload: webhookBytes,
			}
			fromPayloadBytes, _ := json.Marshal(fromPayload)

			req := models.FetchNextPaymentsRequest{
				FromPayload: fromPayloadBytes,
				PageSize:    10,
			}

			m.EXPECT().ListTransactions(gomock.Any(), gomock.Any()).Return(
				client.ListTransactionResponse{},
				errors.New("test error"),
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should fetch payments successfully", func(ctx SpecContext) {
			webhook := client.AccountTransactionsModifiedWebhook{
				ExternalUserID: "user_123",
				Account: struct {
					ID string `json:"id"`
				}{
					ID: "account_1",
				},
				Transactions: client.WebhookTransactions{
					EarliestModifiedBookedDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					LatestModifiedBookedDate:   time.Date(2023, 1, 31, 0, 0, 0, 0, time.UTC),
				},
			}
			webhookBytes, _ := json.Marshal(webhook)

			fromPayload := models.BankBridgeFromPayload{
				FromPayload: webhookBytes,
			}
			fromPayloadBytes, _ := json.Marshal(fromPayload)

			req := models.FetchNextPaymentsRequest{
				FromPayload: fromPayloadBytes,
				PageSize:    10,
			}

			expectedRequest := client.ListTransactionRequest{
				UserID:        "user_123",
				AccountID:     "account_1",
				BookedDateGTE: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				BookedDateLTE: time.Date(2023, 1, 31, 0, 0, 0, 0, time.UTC),
				PageSize:      10,
				NextPageToken: "",
			}

			m.EXPECT().ListTransactions(gomock.Any(), expectedRequest).Return(
				client.ListTransactionResponse{
					Transactions:  sampleTransactions,
					NextPageToken: "",
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(2))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			// Verify payment details - amount should be 100000 for 1000 with scale 2 (1000 * 100)
			Expect(resp.Payments[0].Reference).To(Equal("transaction_1"))
			Expect(resp.Payments[0].Type).To(Equal(models.PAYMENT_TYPE_PAYIN))
			Expect(resp.Payments[0].Status).To(Equal(models.PAYMENT_STATUS_SUCCEEDED))
			Expect(resp.Payments[0].Amount.String()).To(Equal("100000"))
			Expect(resp.Payments[0].Asset).To(Equal("EUR/2"))

			Expect(resp.Payments[1].Reference).To(Equal("transaction_2"))
			Expect(resp.Payments[1].Type).To(Equal(models.PAYMENT_TYPE_PAYOUT))
			Expect(resp.Payments[1].Status).To(Equal(models.PAYMENT_STATUS_PENDING))
			Expect(resp.Payments[1].Amount.String()).To(Equal("50000"))
			Expect(resp.Payments[1].Asset).To(Equal("EUR/2"))
		})

		It("should handle pagination correctly", func(ctx SpecContext) {
			webhook := client.AccountTransactionsModifiedWebhook{
				ExternalUserID: "user_123",
				Account: struct {
					ID string `json:"id"`
				}{
					ID: "account_1",
				},
				Transactions: client.WebhookTransactions{
					EarliestModifiedBookedDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					LatestModifiedBookedDate:   time.Date(2023, 1, 31, 0, 0, 0, 0, time.UTC),
				},
			}
			webhookBytes, _ := json.Marshal(webhook)

			fromPayload := models.BankBridgeFromPayload{
				FromPayload: webhookBytes,
			}
			fromPayloadBytes, _ := json.Marshal(fromPayload)

			req := models.FetchNextPaymentsRequest{
				FromPayload: fromPayloadBytes,
				PageSize:    1,
			}

			// First page - return one transaction with next page token
			// The current implementation breaks when NextPageToken is not empty, so it will only return the first page
			m.EXPECT().ListTransactions(gomock.Any(), gomock.Any()).Return(
				client.ListTransactionResponse{
					Transactions:  sampleTransactions[:1],
					NextPageToken: "next_token",
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			Expect(resp.HasMore).To(BeTrue())

			// Verify state contains next page token
			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.NextPageToken).To(Equal("next_token"))
		})

		It("should handle existing state", func(ctx SpecContext) {
			webhook := client.AccountTransactionsModifiedWebhook{
				ExternalUserID: "user_123",
				Account: struct {
					ID string `json:"id"`
				}{
					ID: "account_1",
				},
				Transactions: client.WebhookTransactions{
					EarliestModifiedBookedDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					LatestModifiedBookedDate:   time.Date(2023, 1, 31, 0, 0, 0, 0, time.UTC),
				},
			}
			webhookBytes, _ := json.Marshal(webhook)

			fromPayload := models.BankBridgeFromPayload{
				FromPayload: webhookBytes,
			}
			fromPayloadBytes, _ := json.Marshal(fromPayload)

			existingState := paymentsState{
				NextPageToken: "existing_token",
			}
			stateBytes, _ := json.Marshal(existingState)

			req := models.FetchNextPaymentsRequest{
				FromPayload: fromPayloadBytes,
				State:       stateBytes,
				PageSize:    10,
			}

			expectedRequest := client.ListTransactionRequest{
				UserID:        "user_123",
				AccountID:     "account_1",
				BookedDateGTE: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				BookedDateLTE: time.Date(2023, 1, 31, 0, 0, 0, 0, time.UTC),
				PageSize:      10,
				NextPageToken: "existing_token",
			}

			m.EXPECT().ListTransactions(gomock.Any(), expectedRequest).Return(
				client.ListTransactionResponse{
					Transactions:  sampleTransactions,
					NextPageToken: "",
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(2))
		})

		It("should handle invalid from payload", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				FromPayload: json.RawMessage(`invalid json`),
				PageSize:    10,
			}

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should handle invalid webhook payload", func(ctx SpecContext) {
			fromPayload := models.BankBridgeFromPayload{
				FromPayload: json.RawMessage(`invalid json`),
			}
			fromPayloadBytes, _ := json.Marshal(fromPayload)

			req := models.FetchNextPaymentsRequest{
				FromPayload: fromPayloadBytes,
				PageSize:    10,
			}

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should handle invalid state", func(ctx SpecContext) {
			webhook := client.AccountTransactionsModifiedWebhook{
				ExternalUserID: "user_123",
				Account: struct {
					ID string `json:"id"`
				}{
					ID: "account_1",
				},
				Transactions: client.WebhookTransactions{
					EarliestModifiedBookedDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					LatestModifiedBookedDate:   time.Date(2023, 1, 31, 0, 0, 0, 0, time.UTC),
				},
			}
			webhookBytes, _ := json.Marshal(webhook)

			fromPayload := models.BankBridgeFromPayload{
				FromPayload: webhookBytes,
			}
			fromPayloadBytes, _ := json.Marshal(fromPayload)

			req := models.FetchNextPaymentsRequest{
				FromPayload: fromPayloadBytes,
				State:       json.RawMessage(`invalid json`),
				PageSize:    10,
			}

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should handle invalid amount scale", func(ctx SpecContext) {
			webhook := client.AccountTransactionsModifiedWebhook{
				ExternalUserID: "user_123",
				Account: struct {
					ID string `json:"id"`
				}{
					ID: "account_1",
				},
				Transactions: client.WebhookTransactions{
					EarliestModifiedBookedDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					LatestModifiedBookedDate:   time.Date(2023, 1, 31, 0, 0, 0, 0, time.UTC),
				},
			}
			webhookBytes, _ := json.Marshal(webhook)

			fromPayload := models.BankBridgeFromPayload{
				FromPayload: webhookBytes,
			}
			fromPayloadBytes, _ := json.Marshal(fromPayload)

			req := models.FetchNextPaymentsRequest{
				FromPayload: fromPayloadBytes,
				PageSize:    10,
			}

			invalidTransactions := []client.Transaction{
				{
					ID: "transaction_1",
					Amount: client.Amount{
						CurrencyCode: "EUR",
						Value: struct {
							Scale string `json:"scale"`
							Value string `json:"unscaledValue"`
						}{
							Value: "1000",
							Scale: "invalid",
						},
					},
					Status:              "BOOKED",
					AccountID:           "account_1",
					TransactionDateTime: time.Now().UTC(),
					BookedDateTime:      time.Now().UTC(),
					ValueDateTime:       time.Now().UTC(),
					Descriptions: client.Descriptions{
						Display: "Payment 1",
					},
				},
			}

			m.EXPECT().ListTransactions(gomock.Any(), gomock.Any()).Return(
				client.ListTransactionResponse{
					Transactions:  invalidTransactions,
					NextPageToken: "",
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should handle unknown transaction status", func(ctx SpecContext) {
			webhook := client.AccountTransactionsModifiedWebhook{
				ExternalUserID: "user_123",
				Account: struct {
					ID string `json:"id"`
				}{
					ID: "account_1",
				},
				Transactions: client.WebhookTransactions{
					EarliestModifiedBookedDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					LatestModifiedBookedDate:   time.Date(2023, 1, 31, 0, 0, 0, 0, time.UTC),
				},
			}
			webhookBytes, _ := json.Marshal(webhook)

			fromPayload := models.BankBridgeFromPayload{
				FromPayload: webhookBytes,
			}
			fromPayloadBytes, _ := json.Marshal(fromPayload)

			req := models.FetchNextPaymentsRequest{
				FromPayload: fromPayloadBytes,
				PageSize:    10,
			}

			unknownStatusTransactions := []client.Transaction{
				{
					ID: "transaction_1",
					Amount: client.Amount{
						CurrencyCode: "EUR",
						Value: struct {
							Scale string `json:"scale"`
							Value string `json:"unscaledValue"`
						}{
							Value: "1000",
							Scale: "2",
						},
					},
					Status:              "UNKNOWN_STATUS",
					AccountID:           "account_1",
					TransactionDateTime: time.Now().UTC(),
					BookedDateTime:      time.Now().UTC(),
					ValueDateTime:       time.Now().UTC(),
					Descriptions: client.Descriptions{
						Display: "Payment 1",
					},
				},
			}

			m.EXPECT().ListTransactions(gomock.Any(), gomock.Any()).Return(
				client.ListTransactionResponse{
					Transactions:  unknownStatusTransactions,
					NextPageToken: "",
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			Expect(int(resp.Payments[0].Status)).To(Equal(100))
		})
	})
})
