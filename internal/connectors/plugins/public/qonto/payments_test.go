package qonto

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/formancehq/go-libs/v3/logging"
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
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{
				client: m,
				logger: logging.NewDefaultLogger(GinkgoWriter, true, false, false),
			}
			pageSize = 50
			from, _ = json.Marshal(models.PSPAccount{Reference: "asdf"})

			sampleTransactions = generateTestSampleTransactions()
		})

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

	})
	// add test for missing payload
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
		assertTransactionsMapping(transactionsUsed[i], transaction)
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

func generateTestSampleTransactions() (sampleTransactions []client.Transactions) {
	sampleTransactions = make([]client.Transactions, 0)
	for i := 0; i < 20; i++ {
		sampleTransactions = append(sampleTransactions, client.Transactions{
			Reference: fmt.Sprintf("transaction-%d", i),
			Status:    "active",
			UpdatedAt: fmt.Sprintf("2021-01-01T00:%02d:00.001Z", i),
		})
	}

	return
}

func assertTransactionsMapping(transaction client.Transactions, resultingPSPAPayment models.PSPPayment) {
	Expect(resultingPSPAPayment.Reference).To(Equal(transaction.Id))
}

//func assertTransactionMapping(beneficiary client.Transactions, resultingPSPAccount models.PSPPayment) {
//	counter, err := strconv.Atoi(beneficiary.ID)
//	Expect(err).To(BeNil())
//
//	expectedReference := ""
//	expectedCurrency := ""
//	switch counter % 3 {
//	case 0:
//		expectedCurrency = "EUR/2"
//		expectedReference = beneficiary.BankAccount.Iban + "-" + beneficiary.BankAccount.Bic
//	case 1:
//		expectedCurrency = "GBP/2"
//		expectedReference = beneficiary.BankAccount.AccountNUmber + "-" + beneficiary.BankAccount.SwiftSortCode + "-" + beneficiary.BankAccount.IntermediaryBankBic
//	case 2:
//		expectedCurrency = "USD/2"
//		expectedReference = beneficiary.BankAccount.AccountNUmber + "-" + beneficiary.BankAccount.RoutingNumber + "-" + beneficiary.BankAccount.IntermediaryBankBic
//	}
//	Expect(resultingPSPAccount.Reference).To(Equal(expectedReference))
//	Expect(*resultingPSPAccount.Name).To(Equal(beneficiary.Name))
//	Expect(resultingPSPAccount.CreatedAt.Format(client.QONTO_TIMEFORMAT)).To(Equal(beneficiary.CreatedAt))
//	Expect(*resultingPSPAccount.DefaultAsset).To(Equal(expectedCurrency))
//	Expect(resultingPSPAccount.Metadata).To(Equal(map[string]string{
//		"beneficiary_id":                     beneficiary.ID,
//		"bank_account_number":                beneficiary.BankAccount.AccountNUmber,
//		"bank_account_iban":                  beneficiary.BankAccount.Iban,
//		"bank_account_bic":                   beneficiary.BankAccount.Bic,
//		"bank_account_swift_sort_code":       beneficiary.BankAccount.SwiftSortCode,
//		"bank_account_routing_number":        beneficiary.BankAccount.RoutingNumber,
//		"bank_account_intermediary_bank_bic": beneficiary.BankAccount.IntermediaryBankBic,
//		"updated_at":                         beneficiary.UpdatedAt,
//	}))
//}
