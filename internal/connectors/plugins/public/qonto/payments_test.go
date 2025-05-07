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
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Qonto *Plugin Payments", func() {
	Context("fetch next payments", func() {
		var (
			plg                *Plugin
			m                  *client.MockClient
			pageSize           int
			from               []byte
			sampleTransactions []client.Transactions
			sampleTransaction  client.Transactions
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
			It("should return an error - get transactions error", func(ctx SpecContext) {
				// Given a valid request but the client fails
				req := models.FetchNextPaymentsRequest{
					State:       []byte(`{}`),
					PageSize:    pageSize,
					FromPayload: from,
				}

				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
					nil,
					errors.New("test error"),
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsErrorResponse(resp, err, errors.New("test error"))
			})

			It("should return an error - missing pageSize in request", func(ctx SpecContext) {
				// Given a request with missing pageSize
				req := models.FetchNextPaymentsRequest{
					State:       []byte(`{}`),
					FromPayload: from,
				}

				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0).Return(
					sampleTransactions,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsErrorResponse(resp, err, errors.New("invalid request, missing page size in request"))
			})

			It("should return an error - missing FromPayload in request", func(ctx SpecContext) {
				// Given a request with missing pageSize
				req := models.FetchNextPaymentsRequest{
					State:    []byte(`{}`),
					PageSize: pageSize,
				}

				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0).Return(
					sampleTransactions,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsErrorResponse(resp, err, errors.New("missing from payload in request"))
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
			m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
				transactionsReturnedByClient,
				nil,
			)

			// When
			resp, err := plg.FetchNextPayments(ctx, req)

			// Then
			assertTransactionsSuccessResponse(resp, err, transactionsReturnedByClient, 1, false)
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
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(resp, err, transactionsReturnedByClient, 1, false)
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
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(resp, err, transactionsReturnedByClient, 1, false)
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
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(resp, err, transactionsReturnedByClient, 1, false)
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
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(resp, err, transactionsReturnedByClient, 1, false)
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
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(resp, err, transactionsReturnedByClient, 1, false)
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
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(resp, err, transactionsReturnedByClient, 1, false)
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

				sampleTransaction.Status = "pending"
				transactionsReturnedByClient := []client.Transactions{sampleTransaction}
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(resp, err, transactionsReturnedByClient, 1, false)
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
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(resp, err, transactionsReturnedByClient, 1, false)
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
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(resp, err, transactionsReturnedByClient, 1, false)
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
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(resp, err, transactionsReturnedByClient, 1, false)
				Expect(resp.Payments[0].Status).To(Equal(models.PAYMENT_STATUS_UNKNOWN))
			})
		})

		Describe("pagination", func() {
			var (
				sampleTransactions     []client.Transactions
				transactionsToGenerate int
			)
			BeforeEach(func() {
				transactionsToGenerate = 20
				sampleTransactions = generateTestSampleTransactions(transactionsToGenerate)
			})
			It("should not return more than pageSize", func(ctx SpecContext) {

				// Given a valid request
				req := models.FetchNextPaymentsRequest{
					State:       []byte(`{}`),
					PageSize:    5,
					FromPayload: from,
				}

				transactionsReturnedByClient := sampleTransactions
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), 1, 5).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(resp, err, transactionsReturnedByClient[:5], 1, true)
			})
			It("should ignore already processed transactions", func(ctx SpecContext) {
				// Given a valid request
				req := models.FetchNextPaymentsRequest{
					State:       []byte(fmt.Sprintf(`{"lastUpdatedAt": "%v", "lastPage": 1}`, sampleTransactions[9].UpdatedAt)),
					PageSize:    pageSize,
					FromPayload: from,
				}

				transactionsReturnedByClient := sampleTransactions
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), 1, pageSize).Return(
					transactionsReturnedByClient,
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(resp, err, transactionsReturnedByClient[10:20], 1, false)
			})
			It("should return hasMore=true when some more txns are present", func(ctx SpecContext) {
				// Given a valid request
				req := models.FetchNextPaymentsRequest{
					State:       []byte(fmt.Sprintf(`{"lastUpdatedAt": "%v", "lastPage": 2}`, sampleTransactions[9].UpdatedAt)),
					PageSize:    5,
					FromPayload: from,
				}

				transactionsReturnedByClient := sampleTransactions
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), 2, 5).Times(1).Return(
					transactionsReturnedByClient[5:10],
					nil,
				)
				m.EXPECT().GetTransactions(gomock.Any(), gomock.Any(), gomock.Any(), 3, 5).Times(1).Return(
					transactionsReturnedByClient[10:15],
					nil,
				)

				// When
				resp, err := plg.FetchNextPayments(ctx, req)

				// Then
				assertTransactionsSuccessResponse(resp, err, transactionsReturnedByClient[10:15], 3, true)
			})
		})
	})
})

func assertTransactionsErrorResponse(resp models.FetchNextPaymentsResponse, err error, expectedError error) {
	Expect(err).ToNot(BeNil())
	Expect(err).To(MatchError(expectedError))
	Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
}

func assertTransactionsSuccessResponse(
	resp models.FetchNextPaymentsResponse,
	err error,
	transactionsUsed []client.Transactions,
	lastPage int,
	hasMore bool,
) {
	Expect(err).To(BeNil())
	Expect(resp.Payments).To(HaveLen(len(transactionsUsed)))
	for i, transaction := range resp.Payments {
		assertSimpleTransactionsMapping(transactionsUsed[i], transaction)
	}

	var expectedLastUpdatedAt time.Time
	if len(transactionsUsed) == 0 {
		expectedLastUpdatedAt = time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC)
	} else {
		expectedLastUpdatedAt, _ = time.ParseInLocation(
			client.QONTO_TIMEFORMAT,
			transactionsUsed[len(transactionsUsed)-1].UpdatedAt,
			time.UTC,
		)
	}

	expectedState := paymentsState{
		LastUpdatedAt: expectedLastUpdatedAt,
		LastPage:      lastPage,
	}

	var actualState paymentsState
	err = json.Unmarshal(resp.NewState, &actualState)
	Expect(err).To(BeNil())
	Expect(actualState.LastUpdatedAt).To(Equal(expectedState.LastUpdatedAt))
	Expect(actualState.LastPage).To(Equal(expectedState.LastPage))
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
	Expect(resultingPSPAPayment.CreatedAt.Format(client.QONTO_TIMEFORMAT)).To(Equal(transaction.EmittedAt))
	Expect(resultingPSPAPayment.Asset).To(Equal("EUR/2"))
	Expect(resultingPSPAPayment.SourceAccountReference).To(Equal(pointer.For(transaction.BankAccountId)))
	Expect(resultingPSPAPayment.Metadata).To(Equal(map[string]string{
		"updated_at": transaction.UpdatedAt,
	}))
	Expect(resultingPSPAPayment.Raw).To(Equal(expectedRaw))
}
