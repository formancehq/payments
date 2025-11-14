package qonto

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/qonto/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "github.com/golang/mock/gomock"
)

var _ = Describe("Qonto *Plugin Payments", func() {
	Context("fetch next payments", func() {
		var (
			plg               *Plugin
			m                 *client.MockClient
			pageSize          int
			from              []byte
			sampleTransaction client.Transactions
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{
				client: m,
				logger: logging.NewDefaultLogger(GinkgoWriter, true, false, false),
			}
			pageSize = 50
			from, _ = json.Marshal(models.PSPAccount{Reference: "bankAccountId"})
		})

		Describe("Error cases", func() {
			It("get transactions error", func(ctx SpecContext) {
				// Given a valid request but the client fails
				req := models.FetchNextPaymentsRequest{
					State:       []byte(`{}`),
					PageSize:    pageSize,
					FromPayload: from,
				}

				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
					nil,
					errors.New("test error"),
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsErrorResponse(resp, err, errors.New("test error"))
			})

			It("missing pageSize in request", func(ctx SpecContext) {
				// Given a request with missing pageSize
				req := models.FetchNextPaymentsRequest{
					State:       []byte(`{}`),
					FromPayload: from,
				}

				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsErrorResponse(resp, err, errors.New("invalid request, missing page size in request"))
			})

			It("missing FromPayload in request", func(ctx SpecContext) {
				// Given a request with missing pageSize
				req := models.FetchNextPaymentsRequest{
					State:    []byte(`{}`),
					PageSize: pageSize,
				}

				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsErrorResponse(resp, err, errors.New("missing from payload in request"))
			})

			It("invalid FromPayload in request", func(ctx SpecContext) {
				// Given a request with missing pageSize
				req := models.FetchNextPaymentsRequest{
					State:       []byte(`{}`),
					PageSize:    pageSize,
					FromPayload: []byte(`{toto: "tata"}`),
				}

				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsErrorResponse(resp, err, errors.New("failed to unmarshall FromPayload"))
			})

			It("invalid state", func(ctx SpecContext) {
				// Given a request with missing pageSize
				req := models.FetchNextPaymentsRequest{
					State:       []byte(`{toto: "tata"}`),
					PageSize:    pageSize,
					FromPayload: from,
				}

				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsErrorResponse(resp, err, errors.New("failed to unmarshall state"))
			})

			It("invalid transaction emittedAt", func(ctx SpecContext) {
				// Given a valid request that returns a transaction with invalid createdAt
				sampleTransaction = generateSampleTransaction(0)
				req := models.FetchNextPaymentsRequest{
					State:       []byte(`{}`),
					PageSize:    pageSize,
					FromPayload: from,
				}

				sampleTransaction.SubjectType = "Card"
				sampleTransaction.EmittedAt = "invalid"
				transactionsReturnedByClient := []client.Transactions{sampleTransaction}
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsErrorResponse(resp, err, errors.New("invalid time format for emittedAt transaction"))
			})

			It("invalid transaction updatedAt", func(ctx SpecContext) {
				// Given a valid request that returns a transaction with invalid createdAt
				sampleTransaction = generateSampleTransaction(0)
				req := models.FetchNextPaymentsRequest{
					State:       []byte(`{}`),
					PageSize:    pageSize,
					FromPayload: from,
				}

				sampleTransaction.SubjectType = "Card"
				sampleTransaction.UpdatedAt = "invalid"
				transactionsReturnedByClient := []client.Transactions{sampleTransaction}
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsErrorResponse(resp, err, errors.New("invalid time format for updatedAt transaction"))
			})
		})

		It("should fetch transactions - no state no results from client", func(ctx SpecContext) {
			// Given a valid request but the client doesn't have results
			req := models.FetchNextPaymentsRequest{
				State:       []byte(`{}`),
				PageSize:    pageSize,
				FromPayload: from,
			}

			transactionsReturnedByClient := make([]client.Transactions, 0)
			m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
				transactionsReturnedByClient,
				nil,
			)

			// When
			resp, err := plg.FetchNextPayments(ctx, req)

			// Then
			assertTransactionsSuccessResponse(
				resp,
				err,
				client.TransactionStatusPending,
				client.TransactionStatusDeclined,
				transactionsReturnedByClient,
				1,
				true,
			)
		})

		Describe("transaction to payment mapping", func() {
			BeforeEach(func() {
				sampleTransaction = generateSampleTransaction(0)
			})
			It("simple transaction", func(ctx SpecContext) {
				// Given a valid request, with a card transaction
				req := models.FetchNextPaymentsRequest{
					State:       []byte(`{}`),
					PageSize:    pageSize,
					FromPayload: from,
				}

				sampleTransaction.SubjectType = "Card"
				transactionsReturnedByClient := []client.Transactions{sampleTransaction}
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(
					resp,
					err,
					client.TransactionStatusPending,
					client.TransactionStatusDeclined,
					transactionsReturnedByClient,
					1,
					true,
				)
				Expect(resp.Payments[0].ParentReference).To(Equal(sampleTransaction.Id))
				Expect(resp.Payments[0].Type).To(Equal(models.PAYMENT_TYPE_PAYOUT))
				Expect(resp.Payments[0].Scheme).To(Equal(models.PAYMENT_SCHEME_UNKNOWN))
				Expect(resp.Payments[0].DestinationAccountReference).To(BeNil())
			})
			It("should map a transfer transaction to a payment with dest account", func(ctx SpecContext) {

				// Given a valid request, with a transfer transaction
				req := models.FetchNextPaymentsRequest{
					State:       []byte(`{}`),
					PageSize:    pageSize,
					FromPayload: from,
				}

				sampleTransaction.SubjectType = "Transfer"
				sampleTransaction.Transfer = &client.CounterpartyDetails{
					CounterpartyAccountNumber:        "IBAN",
					CounterpartyAccountNumberFormat:  "iban",
					CounterpartyBankIdentifier:       "BIC",
					CounterpartyBankIdentifierFormat: "bic",
				}
				transactionsReturnedByClient := []client.Transactions{sampleTransaction}
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(
					resp,
					err,
					client.TransactionStatusPending,
					client.TransactionStatusDeclined,
					transactionsReturnedByClient,
					1,
					true,
				)
				Expect(resp.Payments[0].Type).To(Equal(models.PAYMENT_TYPE_PAYOUT))
				Expect(resp.Payments[0].Scheme).To(Equal(models.PAYMENT_SCHEME_UNKNOWN))
				Expect(resp.Payments[0].DestinationAccountReference).To(Equal(pointer.For("IBAN-BIC")))
			})
			It("should map a direct debit transaction to a payment with dest account", func(ctx SpecContext) {
				// Given a valid request, with a direct debit transaction
				req := models.FetchNextPaymentsRequest{
					State:       []byte(`{}`),
					PageSize:    pageSize,
					FromPayload: from,
				}

				sampleTransaction.SubjectType = "DirectDebit"
				sampleTransaction.DirectDebit = &client.CounterpartyDetails{
					CounterpartyAccountNumber:        "IBAN",
					CounterpartyAccountNumberFormat:  "iban",
					CounterpartyBankIdentifier:       "BIC",
					CounterpartyBankIdentifierFormat: "bic",
				}
				transactionsReturnedByClient := []client.Transactions{sampleTransaction}
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(
					resp,
					err,
					client.TransactionStatusPending,
					client.TransactionStatusDeclined,
					transactionsReturnedByClient,
					1,
					true,
				)
				Expect(resp.Payments[0].Type).To(Equal(models.PAYMENT_TYPE_PAYOUT))
				Expect(resp.Payments[0].Scheme).To(Equal(models.PAYMENT_SCHEME_SEPA_DEBIT))
				Expect(resp.Payments[0].DestinationAccountReference).To(Equal(pointer.For("IBAN-BIC")))
			})
			It("should map a direct debit collection transaction to a payment with dest account", func(ctx SpecContext) {

				// Given a valid request, with a direct debit collection transaction
				req := models.FetchNextPaymentsRequest{
					State:       []byte(`{}`),
					PageSize:    pageSize,
					FromPayload: from,
				}

				sampleTransaction.SubjectType = "DirectDebitCollection"
				sampleTransaction.DirectDebitCollection = &client.CounterpartyDetails{
					CounterpartyAccountNumber:        "IBAN",
					CounterpartyAccountNumberFormat:  "iban",
					CounterpartyBankIdentifier:       "BIC",
					CounterpartyBankIdentifierFormat: "bic",
				}
				transactionsReturnedByClient := []client.Transactions{sampleTransaction}
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(
					resp,
					err,
					client.TransactionStatusPending,
					client.TransactionStatusDeclined,
					transactionsReturnedByClient,
					1,
					true,
				)
				Expect(resp.Payments[0].Type).To(Equal(models.PAYMENT_TYPE_PAYIN))
				Expect(resp.Payments[0].Scheme).To(Equal(models.PAYMENT_SCHEME_SEPA_CREDIT))
				Expect(resp.Payments[0].DestinationAccountReference).To(Equal(pointer.For("IBAN-BIC")))
			})
			It("should map an income transaction to a payment with dest account", func(ctx SpecContext) {

				// Given a valid request, with an income transaction
				req := models.FetchNextPaymentsRequest{
					State:       []byte(`{}`),
					PageSize:    pageSize,
					FromPayload: from,
				}

				sampleTransaction.SubjectType = "Income"
				sampleTransaction.Income = &client.CounterpartyDetails{
					CounterpartyAccountNumber:        "IBAN",
					CounterpartyAccountNumberFormat:  "iban",
					CounterpartyBankIdentifier:       "BIC",
					CounterpartyBankIdentifierFormat: "bic",
				}
				transactionsReturnedByClient := []client.Transactions{sampleTransaction}
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(
					resp,
					err,
					client.TransactionStatusPending,
					client.TransactionStatusDeclined,
					transactionsReturnedByClient,
					1,
					true,
				)
				Expect(resp.Payments[0].Type).To(Equal(models.PAYMENT_TYPE_PAYIN))
				Expect(resp.Payments[0].Scheme).To(Equal(models.PAYMENT_SCHEME_UNKNOWN))
				Expect(resp.Payments[0].DestinationAccountReference).To(Equal(pointer.For("IBAN-BIC")))
			})
			It("should map a swift income transaction to a payment with dest account", func(ctx SpecContext) {

				// Given a valid request, with an income transaction
				req := models.FetchNextPaymentsRequest{
					State:       []byte(`{}`),
					PageSize:    pageSize,
					FromPayload: from,
				}

				sampleTransaction.SubjectType = "Income"
				sampleTransaction.Income = &client.CounterpartyDetails{
					CounterpartyAccountNumber:        "ACCOUNT-NUMBER",
					CounterpartyAccountNumberFormat:  "unstructured",
					CounterpartyBankIdentifier:       "SORT-CODE",
					CounterpartyBankIdentifierFormat: "sort_code",
				}
				transactionsReturnedByClient := []client.Transactions{sampleTransaction}
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(
					resp,
					err,
					client.TransactionStatusPending,
					client.TransactionStatusDeclined,
					transactionsReturnedByClient,
					1,
					true,
				)
				Expect(resp.Payments[0].Type).To(Equal(models.PAYMENT_TYPE_PAYIN))
				Expect(resp.Payments[0].Scheme).To(Equal(models.PAYMENT_SCHEME_UNKNOWN))
				Expect(resp.Payments[0].DestinationAccountReference).To(Equal(pointer.For("ACCOUNT-NUMBER-SORT-CODE")))
			})
			It("should map a pending transaction to the right status", func(ctx SpecContext) {
				// Given a valid request, with a pending transaction
				req := models.FetchNextPaymentsRequest{
					State:       []byte(`{}`),
					PageSize:    pageSize,
					FromPayload: from,
				}

				sampleTransaction.Status = client.TransactionStatusPending
				transactionsReturnedByClient := []client.Transactions{sampleTransaction}
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(
					resp,
					err,
					client.TransactionStatusPending,
					client.TransactionStatusDeclined,
					transactionsReturnedByClient,
					1,
					true,
				)
				Expect(resp.Payments[0].Status).To(Equal(models.PAYMENT_STATUS_PENDING))
			})
			It("should map a declined transaction to the right status", func(ctx SpecContext) {
				// Given a valid request, with a declined transaction
				req := models.FetchNextPaymentsRequest{
					State:       []byte(`{}`),
					PageSize:    pageSize,
					FromPayload: from,
				}

				sampleTransaction.Status = "declined"
				transactionsReturnedByClient := []client.Transactions{sampleTransaction}
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(
					resp,
					err,
					client.TransactionStatusPending,
					client.TransactionStatusDeclined,
					transactionsReturnedByClient,
					1,
					true,
				)
				Expect(resp.Payments[0].Status).To(Equal(models.PAYMENT_STATUS_FAILED))
			})
			It("should map a completed transaction to the right status", func(ctx SpecContext) {
				// Given a valid request, with a completed transaction
				req := models.FetchNextPaymentsRequest{
					State:       []byte(`{}`),
					PageSize:    pageSize,
					FromPayload: from,
				}

				sampleTransaction.Status = "completed"
				transactionsReturnedByClient := []client.Transactions{sampleTransaction}
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(
					resp,
					err,
					client.TransactionStatusPending,
					client.TransactionStatusDeclined,
					transactionsReturnedByClient,
					1,
					true,
				)
				Expect(resp.Payments[0].Status).To(Equal(models.PAYMENT_STATUS_SUCCEEDED))
			})
			It("should map an unknown status transaction to the right status", func(ctx SpecContext) {
				// Given a valid request, with an unknown status transaction
				req := models.FetchNextPaymentsRequest{
					State:       []byte(`{}`),
					PageSize:    pageSize,
					FromPayload: from,
				}

				sampleTransaction.Status = "toto"
				transactionsReturnedByClient := []client.Transactions{sampleTransaction}
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(
					resp,
					err,
					client.TransactionStatusPending,
					client.TransactionStatusDeclined,
					transactionsReturnedByClient,
					1,
					true,
				)
				Expect(resp.Payments[0].Status).To(Equal(models.PAYMENT_STATUS_UNKNOWN))
			})
		})

		Describe("pagination", func() {
			var (
				sampleTransactions []client.Transactions
			)
			BeforeEach(func() {
				sampleTransactions = generateTestSampleTransactions(20)
			})

			It("pageSize is ignored if the API returns more than the expected count", func(ctx SpecContext) {

				// Given a valid request
				req := models.FetchNextPaymentsRequest{
					State:       []byte(`{}`),
					PageSize:    5,
					FromPayload: from,
				}

				transactionsReturnedByClient := sampleTransactions
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), 5).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(
					resp,
					err,
					client.TransactionStatusPending,
					client.TransactionStatusPending,
					transactionsReturnedByClient,
					1,
					true,
				)
			})
			It("should ignore already processed transactions", func(ctx SpecContext) {
				// Given a valid request
				req := models.FetchNextPaymentsRequest{
					State: []byte(fmt.Sprintf(
						`{"lastUpdatedAt": {"pending": "%v"}, "transactionStatusToFetch": "pending", "lastProcessedId": {"pending": "%v"}, "page": {"pending": 1}}`,
						sampleTransactions[9].UpdatedAt,
						sampleTransactions[9].Id,
					)),
					PageSize:    pageSize,
					FromPayload: from,
				}

				transactionsReturnedByClient := sampleTransactions
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), pageSize).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(
					resp,
					err,
					client.TransactionStatusPending,
					client.TransactionStatusDeclined,
					transactionsReturnedByClient[10:20],
					1,
					true,
				)
			})

			It("should return hasMore=true when some more txns are present in the same status", func(ctx SpecContext) {
				// Given a valid request
				req := models.FetchNextPaymentsRequest{
					State: []byte(fmt.Sprintf(
						`{"lastUpdatedAt": {"pending": "%v"}, "transactionStatusToFetch": "pending"}`,
						sampleTransactions[9].UpdatedAt,
					)),
					PageSize:    5,
					FromPayload: from,
				}

				transactionsReturnedByClient := sampleTransactions
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), 5).Times(1).Return(
					transactionsReturnedByClient[10:15],
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(
					resp,
					err,
					client.TransactionStatusPending,
					client.TransactionStatusDeclined,
					transactionsReturnedByClient[10:15],
					1,
					true,
				)

			})
			// Note -- transition from Pending to Declined already tested as part of the mapping tests
			It("transition from Declined to Completed status", func(ctx SpecContext) {
				// Given a valid request
				req := models.FetchNextPaymentsRequest{
					State: []byte(fmt.Sprintf(
						`{"lastUpdatedAt": {"pending": "%v", "declined": "%v"}, "transactionStatusToFetch": "declined"}`,
						sampleTransactions[9].UpdatedAt,
						sampleTransactions[9].UpdatedAt,
					)),
					PageSize:    pageSize,
					FromPayload: from,
				}

				transactionsReturnedByClient := sampleTransactions[15:20]
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), pageSize).Times(1).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(
					resp,
					err,
					client.TransactionStatusDeclined,
					client.TransactionStatusCompleted,
					transactionsReturnedByClient,
					1,
					true,
				)
			})
			It("transition from Completed back to Pending status", func(ctx SpecContext) {
				// Given a valid request
				req := models.FetchNextPaymentsRequest{
					State: []byte(fmt.Sprintf(
						`{"lastUpdatedAt": {"pending": "%v", "declined": "%v", "completed": "%v"}, "transactionStatusToFetch": "completed"}`,
						sampleTransactions[1].UpdatedAt,
						sampleTransactions[2].UpdatedAt,
						sampleTransactions[3].UpdatedAt,
					)),
					PageSize:    pageSize,
					FromPayload: from,
				}

				transactionsReturnedByClient := sampleTransactions[15:20]
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), pageSize).Times(1).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(
					resp,
					err,
					client.TransactionStatusCompleted,
					client.TransactionStatusPending,
					transactionsReturnedByClient,
					1,
					false,
				)
			})

			It("when multiple payments have the same updatedAt, it should use lastProcessedId to skip already processed payments", func(ctx SpecContext) {
				// Given a valid request with lastProcessedId set
				req := models.FetchNextPaymentsRequest{
					State: []byte(fmt.Sprintf(
						`{"lastUpdatedAt": {"pending": "%v"}, "transactionStatusToFetch": "pending", "lastProcessedId": {"pending": "%v"}, "page": {"pending": 1}}`,
						sampleTransactions[4].UpdatedAt,
						sampleTransactions[4].Id,
					)),
					PageSize:    5,
					FromPayload: from,
				}

				// Set all transactions to have the same updatedAt
				for i := range sampleTransactions {
					sampleTransactions[i].UpdatedAt = sampleTransactions[4].UpdatedAt
				}

				transactionsReturnedByClient := sampleTransactions[5:10]
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), 5).Times(1).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				// We expect transactions [5:10] to be returned
				// The page should be incremented to 2
				assertTransactionsSuccessResponse(
					resp,
					err,
					client.TransactionStatusPending,
					client.TransactionStatusDeclined,
					transactionsReturnedByClient,
					2,
					true,
				)
			})
		})
	})
})

func assertTransactionsErrorResponse(resp models.FetchNextPaymentsResponse, err error, expectedError error) {
	Expect(err).ToNot(BeNil())
	Expect(err).To(MatchError(ContainSubstring(expectedError.Error())))
	Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
}

func assertTransactionsSuccessResponse(
	resp models.FetchNextPaymentsResponse,
	err error,
	fetchedTransactionStatus string,
	nextStateTransactionStatus string,
	transactionsUsed []client.Transactions,
	nextStateExpectedPage int,
	hasMore bool,
) {
	Expect(err).To(BeNil())
	Expect(resp.Payments).To(HaveLen(len(transactionsUsed)))
	for i, transaction := range resp.Payments {
		assertSimpleTransactionsMapping(transactionsUsed[i], transaction)
	}

	var expectedLastUpdatedAt map[string]time.Time
	if len(transactionsUsed) == 0 {
		expectedLastUpdatedAt = map[string]time.Time{}
	} else {
		timeExpected, _ := time.ParseInLocation(
			client.QontoTimeformat,
			transactionsUsed[len(transactionsUsed)-1].UpdatedAt,
			time.UTC,
		)
		expectedLastUpdatedAt = map[string]time.Time{
			fetchedTransactionStatus: timeExpected,
		}
	}

	var expectedLastProcessedId map[string]string
	var expectedPage map[string]int
	if len(transactionsUsed) == 0 {
		expectedLastProcessedId = map[string]string{}
		expectedPage = map[string]int{
			fetchedTransactionStatus: nextStateExpectedPage,
		}
	} else {
		expectedLastProcessedId = map[string]string{
			fetchedTransactionStatus: transactionsUsed[len(transactionsUsed)-1].Id,
		}
		expectedPage = map[string]int{
			fetchedTransactionStatus: nextStateExpectedPage,
		}
	}

	expectedState := paymentsState{
		LastUpdatedAt:            expectedLastUpdatedAt,
		TransactionStatusToFetch: nextStateTransactionStatus,
		LastProcessedId:          expectedLastProcessedId,
		Page:                     expectedPage,
	}

	var actualState paymentsState
	err = json.Unmarshal(resp.NewState, &actualState)
	Expect(err).To(BeNil())
	Expect(actualState.LastUpdatedAt[fetchedTransactionStatus]).To(Equal(expectedState.LastUpdatedAt[fetchedTransactionStatus]))
	Expect(actualState.LastProcessedId[fetchedTransactionStatus]).To(Equal(expectedState.LastProcessedId[fetchedTransactionStatus]))
	Expect(actualState.Page[fetchedTransactionStatus]).To(Equal(expectedState.Page[fetchedTransactionStatus]))
	Expect(resp.HasMore).To(Equal(hasMore))
}

func generateTestSampleTransactions(transactionToGenerate int) (sampleTransactions []client.Transactions) {
	sampleTransactions = make([]client.Transactions, 0)
	for i := 0; i < transactionToGenerate; i++ {
		transaction := generateSampleTransaction(i)
		sampleTransactions = append(sampleTransactions, transaction)
	}

	return
}

func generateSampleTransaction(i int) client.Transactions {
	var (
		side string
	)
	switch i {
	case 0:
		side = "credit"
	default:
		side = "debit"
	}

	return client.Transactions{
		Id:                  fmt.Sprintf("%d", i),
		TransactionId:       fmt.Sprintf("%d", i),
		Amount:              "20",
		AmountCents:         2000,
		SettledBalance:      "5",
		SettledBalanceCents: 500,
		AttachmentsIds:      pointer.For([]string{"1", "2", "3"}),
		Logo: pointer.For(client.LogoDetails{
			Small:  "small",
			Medium: "medium",
		}),
		LocalAmount:           "10",
		LocalAmountCents:      1000,
		Side:                  side,
		OperationType:         "card",
		Currency:              "EUR",
		LocalCurrency:         "GBP",
		Label:                 "toto",
		CleanCounterpartyName: "toto",
		SettledAt:             "",
		EmittedAt:             fmt.Sprintf("2021-01-01T00:%02d:00.001Z", i),
		UpdatedAt:             fmt.Sprintf("2021-02-01T00:%02d:00.001Z", i),
		Status:                "completed",
		Note:                  "note",
		Reference:             fmt.Sprintf("transaction-%d", i),
		VatAmount:             "0",
		VatAmountCents:        0,
		VatRate:               "0",
		InitiatorId:           "123456",
		LabelIds:              pointer.For([]string{"4", "5", "6"}),
		AttachmentLost:        false,
		AttachmentRequired:    false,
		CardLastDigits:        "123",
		Category:              "456",
		SubjectType:           "Card",
		BankAccountId:         "bankAccountId",
		IsExternalTransaction: false,
	}
}

func assertSimpleTransactionsMapping(transaction client.Transactions, resultingPSPAPayment models.PSPPayment) {
	var expectedRaw json.RawMessage
	expectedRaw, _ = json.Marshal(transaction)

	Expect(resultingPSPAPayment.Reference).To(Equal(transaction.Id))
	Expect(resultingPSPAPayment.Amount).To(Equal(big.NewInt(transaction.AmountCents)))
	Expect(resultingPSPAPayment.CreatedAt.Format(client.QontoTimeformat)).To(Equal(transaction.EmittedAt))
	Expect(resultingPSPAPayment.Asset).To(Equal("EUR/2"))
	Expect(resultingPSPAPayment.SourceAccountReference).To(Equal(pointer.For(transaction.BankAccountId)))
	Expect(resultingPSPAPayment.Metadata).To(Equal(map[string]string{
		"updated_at": transaction.UpdatedAt,
	}))
	Expect(resultingPSPAPayment.Raw).To(Equal(expectedRaw))
}
