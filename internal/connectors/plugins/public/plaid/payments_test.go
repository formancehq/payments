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

var _ = Describe("Plaid *Plugin Payments", func() {
	Context("fetch next payments", func() {
		var (
			ctrl *gomock.Controller
			plg  models.Plugin
			m    *client.MockClient

			sampleTransactions        []plaid.Transaction
			sampleRemovedTransactions []plaid.RemovedTransaction
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{client: m}

			sampleTransactions = make([]plaid.Transaction, 0)
			sampleRemovedTransactions = make([]plaid.RemovedTransaction, 0)
			for i := 0; i < 3; i++ {
				transaction := plaid.NewTransactionWithDefaults()
				transaction.SetTransactionId(fmt.Sprintf("transaction_%d", i))
				transaction.SetAccountId(fmt.Sprintf("account_%d", i))
				transaction.SetAmount(float64(100 + i))
				transaction.SetDate("2023-01-01")
				transaction.SetIsoCurrencyCode("USD")
				sampleTransactions = append(sampleTransactions, *transaction)

				removedTransaction := plaid.NewRemovedTransactionWithDefaults()
				removedTransaction.SetTransactionId(fmt.Sprintf("transaction_%d", i))
				removedTransaction.SetAccountId(fmt.Sprintf("account_%d", i))
				sampleRemovedTransactions = append(sampleRemovedTransactions, *removedTransaction)
			}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should return an error - list transactions error", func(ctx SpecContext) {
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

			req := models.FetchNextPaymentsRequest{
				FromPayload: fromPayloadBytes,
				PageSize:    10,
			}

			m.EXPECT().ListTransactions(gomock.Any(), "test-token", "", 10).Return(
				plaid.TransactionsSyncResponse{},
				errors.New("test error"),
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should fetch payments successfully - no state", func(ctx SpecContext) {
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

			req := models.FetchNextPaymentsRequest{
				FromPayload: fromPayloadBytes,
				PageSize:    10,
			}

			m.EXPECT().ListTransactions(gomock.Any(), "test-token", "", 10).Return(
				plaid.TransactionsSyncResponse{
					Added:      sampleTransactions,
					Modified:   []plaid.Transaction{},
					Removed:    []plaid.RemovedTransaction{},
					HasMore:    false,
					NextCursor: "next-cursor",
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(3))
			Expect(resp.PaymentsToDelete).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastCursor).To(Equal("next-cursor"))
		})

		It("should fetch payments successfully - with state", func(ctx SpecContext) {
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

			state := paymentsState{LastCursor: "previous-cursor"}
			stateBytes, _ := json.Marshal(state)

			req := models.FetchNextPaymentsRequest{
				FromPayload: fromPayloadBytes,
				State:       stateBytes,
				PageSize:    10,
			}

			m.EXPECT().ListTransactions(gomock.Any(), "test-token", "previous-cursor", 10).Return(
				plaid.TransactionsSyncResponse{
					Added:      []plaid.Transaction{},
					Modified:   sampleTransactions,
					Removed:    []plaid.RemovedTransaction{},
					HasMore:    true,
					NextCursor: "new-cursor",
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(3))
			Expect(resp.PaymentsToDelete).To(HaveLen(0))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())

			var newState paymentsState
			err = json.Unmarshal(resp.NewState, &newState)
			Expect(err).To(BeNil())
			Expect(newState.LastCursor).To(Equal("new-cursor"))
		})

		It("should handle removed transactions", func(ctx SpecContext) {
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

			req := models.FetchNextPaymentsRequest{
				FromPayload: fromPayloadBytes,
				PageSize:    10,
			}

			m.EXPECT().ListTransactions(gomock.Any(), "test-token", "", 10).Return(
				plaid.TransactionsSyncResponse{
					Added:      []plaid.Transaction{},
					Modified:   []plaid.Transaction{},
					Removed:    sampleRemovedTransactions,
					HasMore:    false,
					NextCursor: "next-cursor",
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(0))
			Expect(resp.PaymentsToDelete).To(HaveLen(3))
			Expect(resp.PaymentsToDelete[0].Reference).To(Equal("transaction_0"))
		})

		It("should handle empty transactions response", func(ctx SpecContext) {
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

			req := models.FetchNextPaymentsRequest{
				FromPayload: fromPayloadBytes,
				PageSize:    10,
			}

			m.EXPECT().ListTransactions(gomock.Any(), "test-token", "", 10).Return(
				plaid.TransactionsSyncResponse{
					Added:      []plaid.Transaction{},
					Modified:   []plaid.Transaction{},
					Removed:    []plaid.RemovedTransaction{},
					HasMore:    false,
					NextCursor: "",
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(0))
			Expect(resp.PaymentsToDelete).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
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

			req := models.FetchNextPaymentsRequest{
				FromPayload: fromPayloadBytes,
				PageSize:    10,
			}

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should handle invalid state", func(ctx SpecContext) {
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

			req := models.FetchNextPaymentsRequest{
				FromPayload: fromPayloadBytes,
				State:       json.RawMessage(`invalid json`),
				PageSize:    10,
			}

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})
	})
})
