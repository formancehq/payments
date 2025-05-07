package qonto

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/formancehq/go-libs/v3/logging"
	"strconv"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/qonto/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Qonto *Plugin External Accounts", func() {
	Context("fetch next external accounts", func() {
		var (
			plg                 *Plugin
			m                   *client.MockClient
			pageSize            int
			sampleBeneficiaries []client.Beneficiary
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{
				client: m,
				logger: logging.NewDefaultLogger(GinkgoWriter, true, false, false),
			}
			pageSize = 50

			sampleBeneficiaries = generateTestSampleBeneficiaries()
		})

		It("should return an error - get beneficiaries error", func(ctx SpecContext) {
			// Given a valid request but the client fails
			req := models.FetchNextExternalAccountsRequest{
				State:    []byte(`{}`),
				PageSize: pageSize,
			}

			m.EXPECT().GetBeneficiaries(gomock.Any(), gomock.Any(), gomock.Any()).Return(
				nil,
				errors.New("test error"),
			)

			// When
			resp, err := plg.FetchNextExternalAccounts(ctx, req)

			// Then
			assertErrorResponse(resp, err, errors.New("test error"))
		})

		It("should return an error - missing pageSize in request", func(ctx SpecContext) {
			// Given a request with missing pageSize
			req := models.FetchNextExternalAccountsRequest{
				State: []byte(`{}`),
			}

			m.EXPECT().GetBeneficiaries(gomock.Any(), gomock.Any(), gomock.Any()).Times(0).Return(
				sampleBeneficiaries,
				nil,
			)

			// When
			resp, err := plg.FetchNextExternalAccounts(ctx, req)

			// Then
			assertErrorResponse(resp, err, errors.New("invalid request, missing page size in request"))
		})

		It("should fetch next accounts - no state no results from client", func(ctx SpecContext) {
			// Given a valid request but the client doesn't have results
			req := models.FetchNextExternalAccountsRequest{
				State:    []byte(`{}`),
				PageSize: pageSize,
			}

			beneficiariesReturnedByClient := make([]client.Beneficiary, 0)
			m.EXPECT().GetBeneficiaries(gomock.Any(), gomock.Any(), gomock.Any()).Return(
				beneficiariesReturnedByClient,
				nil,
			)

			// When
			resp, err := plg.FetchNextExternalAccounts(ctx, req)

			// Then
			assertSuccessResponse(resp, err, beneficiariesReturnedByClient, false)
		})

		It("should fetch next accounts - nil state, no results from client", func(ctx SpecContext) {
			// Given
			req := models.FetchNextExternalAccountsRequest{
				State:    nil,
				PageSize: pageSize,
			}

			beneficiariesReturnedByClient := make([]client.Beneficiary, 0)
			m.EXPECT().GetBeneficiaries(gomock.Any(), gomock.Any(), gomock.Any()).Return(
				beneficiariesReturnedByClient,
				nil,
			)

			// When
			resp, err := plg.FetchNextExternalAccounts(ctx, req)

			// Then
			assertSuccessResponse(resp, err, beneficiariesReturnedByClient, false)
		})

		It("should fetch next accounts - no state, with results", func(ctx SpecContext) {
			// Given
			req := models.FetchNextExternalAccountsRequest{
				State:    []byte(`{}`),
				PageSize: pageSize,
			}

			beneficiariesReturnedByClient := sampleBeneficiaries
			m.EXPECT().GetBeneficiaries(gomock.Any(), gomock.Any(), gomock.Any()).Return(
				beneficiariesReturnedByClient,
				nil,
			)

			// When
			resp, err := plg.FetchNextExternalAccounts(ctx, req)

			// Then
			assertSuccessResponse(resp, err, beneficiariesReturnedByClient, false)
		})

		It("should fetch next accounts - state set, filters out already processed response", func(ctx SpecContext) {
			// Given
			req := models.FetchNextExternalAccountsRequest{
				State:    []byte(fmt.Sprintf(`{"lastUpdatedAt": "%v"}`, sampleBeneficiaries[9].UpdatedAt)),
				PageSize: pageSize,
			}
			beneficiariesReturnedByClient := sampleBeneficiaries
			m.EXPECT().GetBeneficiaries(gomock.Any(), gomock.Any(), gomock.Any()).Return(
				beneficiariesReturnedByClient,
				nil,
			)

			// When
			resp, err := plg.FetchNextExternalAccounts(ctx, req)

			// Then
			assertSuccessResponse(resp, err, beneficiariesReturnedByClient[10:20], false)
		})

		It("should fetch next accounts - no state and pageSize < total", func(ctx SpecContext) {
			// Given
			req := models.FetchNextExternalAccountsRequest{
				State:    []byte("{}"),
				PageSize: 5,
			}

			beneficiariesReturnedByClient := sampleBeneficiaries[0:5]
			m.EXPECT().GetBeneficiaries(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(
				beneficiariesReturnedByClient,
				nil,
			)

			// When
			resp, err := plg.FetchNextExternalAccounts(ctx, req)

			// Then
			assertSuccessResponse(resp, err, beneficiariesReturnedByClient, true)
		})

		It("should fetch next accounts - set state with lastUpdateAt and pageSize < total", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State:    []byte(fmt.Sprintf(`{"lastUpdatedAt": "%v"}`, sampleBeneficiaries[9].UpdatedAt)),
				PageSize: 5,
			}
			beneficiariesReturnedByClient := sampleBeneficiaries[10:15]
			updatedAtFrom, _ := time.ParseInLocation(client.QONTO_TIMEFORMAT, sampleBeneficiaries[9].UpdatedAt, time.UTC)

			m.EXPECT().GetBeneficiaries(gomock.Any(), updatedAtFrom, 5).Times(1).Return(
				beneficiariesReturnedByClient,
				nil,
			)

			// When
			resp, err := plg.FetchNextExternalAccounts(ctx, req)

			// Then
			assertSuccessResponse(resp, err, beneficiariesReturnedByClient, true)
		})

		It("should fetch next accounts - ignores beneficiaries with invalid bank account", func(ctx SpecContext) {
			// given a beneficiary with an invalid bank account
			req := models.FetchNextExternalAccountsRequest{
				State:    []byte(fmt.Sprintf(`{"lastUpdatedAt": "%v"}`, time.Time{}.Format(client.QONTO_TIMEFORMAT))),
				PageSize: pageSize,
			}
			beneficiariesReturnedByClient := sampleBeneficiaries[0:1]
			beneficiariesReturnedByClient[0].BankAccount.Iban = ""
			m.EXPECT().GetBeneficiaries(gomock.Any(), time.Time{}, pageSize).Times(1).Return(
				beneficiariesReturnedByClient,
				nil,
			)

			// When
			resp, err := plg.FetchNextExternalAccounts(ctx, req)

			// Then
			assertSuccessResponse(resp, err, make([]client.Beneficiary, 0), false)
		})
	})
})

func assertErrorResponse(resp models.FetchNextExternalAccountsResponse, err error, expectedError error) {
	Expect(err).ToNot(BeNil())
	Expect(err).To(MatchError(expectedError))
	Expect(resp).To(Equal(models.FetchNextExternalAccountsResponse{}))
}

func assertSuccessResponse(
	resp models.FetchNextExternalAccountsResponse,
	err error,
	beneficiariesUsed []client.Beneficiary,
	hasMore bool,
) {
	Expect(err).To(BeNil())
	Expect(resp.ExternalAccounts).To(HaveLen(len(beneficiariesUsed)))
	for i, account := range resp.ExternalAccounts {
		assertBeneficiaryMapping(beneficiariesUsed[i], account)
	}

	var expectedLastUpdatedAt time.Time
	if len(beneficiariesUsed) == 0 {
		expectedLastUpdatedAt = time.Time{}
	} else {
		expectedLastUpdatedAt, _ = time.ParseInLocation(
			client.QONTO_TIMEFORMAT,
			beneficiariesUsed[len(beneficiariesUsed)-1].UpdatedAt,
			time.UTC,
		)
	}

	expectedState := externalAccountsState{
		LastUpdatedAt: expectedLastUpdatedAt,
	}

	var actualState externalAccountsState
	err = json.Unmarshal(resp.NewState, &actualState)
	Expect(err).To(BeNil())
	Expect(actualState.LastUpdatedAt).To(Equal(expectedState.LastUpdatedAt))
	Expect(resp.HasMore).To(Equal(hasMore))
}

func generateTestSampleBeneficiaries() (sampleBeneficiaries []client.Beneficiary) {
	sampleBeneficiaries = make([]client.Beneficiary, 0)
	for i := 0; i < 20; i++ {
		var beneficiaryBankAccount client.BeneficiaryBankAccount
		var currency string
		switch i % 3 {
		case 0:
			currency = "EUR"
			beneficiaryBankAccount = client.BeneficiaryBankAccount{
				Currency: currency,
				Iban:     fmt.Sprintf("FR76300060000112345678901%02d", i),
				Bic:      fmt.Sprintf("BNPAFRPP%02d", i),
			}
		case 1:
			currency = "GBP"
			beneficiaryBankAccount = client.BeneficiaryBankAccount{
				Currency:            currency,
				AccountNUmber:       fmt.Sprintf("ACCOUNTNUMBER%02d", i),
				IntermediaryBankBic: fmt.Sprintf("BNPAFRPP%02d", i),
				SwiftSortCode:       fmt.Sprintf("SORTCODE%02d", i),
			}
		case 2:
			currency = "USD"
			beneficiaryBankAccount = client.BeneficiaryBankAccount{
				Currency:            currency,
				AccountNUmber:       fmt.Sprintf("ACCOUNTNUMBER%02d", i),
				IntermediaryBankBic: fmt.Sprintf("BNPAFRPP%02d", i),
				RoutingNumber:       fmt.Sprintf("ROUTINGNUMBER%02d", i),
			}
		}
		sampleBeneficiaries = append(sampleBeneficiaries, client.Beneficiary{
			Id:          strconv.Itoa(i),
			Name:        fmt.Sprintf("Account %d", i),
			Status:      "active",
			Trusted:     false,
			BankAccount: beneficiaryBankAccount,
			CreatedAt:   fmt.Sprintf("2020-01-01T00:%02d:00.001Z", i),
			UpdatedAt:   fmt.Sprintf("2021-01-01T00:%02d:00.001Z", i),
		})
	}

	return
}

func assertBeneficiaryMapping(beneficiary client.Beneficiary, resultingPSPAccount models.PSPAccount) {
	counter, err := strconv.Atoi(beneficiary.Id)
	Expect(err).To(BeNil())
	var expectedRaw json.RawMessage
	expectedRaw, _ = json.Marshal(beneficiary)

	expectedReference := ""
	expectedCurrency := ""
	switch counter % 3 {
	case 0:
		expectedCurrency = "EUR/2"
		expectedReference = beneficiary.BankAccount.Iban + "-" + beneficiary.BankAccount.Bic
	case 1:
		expectedCurrency = "GBP/2"
		expectedReference = beneficiary.BankAccount.AccountNUmber + "-" + beneficiary.BankAccount.SwiftSortCode
	case 2:
		expectedCurrency = "USD/2"
		expectedReference = beneficiary.BankAccount.AccountNUmber + "-" + beneficiary.BankAccount.RoutingNumber
	}
	Expect(resultingPSPAccount.Reference).To(Equal(expectedReference))
	Expect(*resultingPSPAccount.Name).To(Equal(beneficiary.Name))
	Expect(resultingPSPAccount.CreatedAt.Format(client.QONTO_TIMEFORMAT)).To(Equal(beneficiary.CreatedAt))
	Expect(*resultingPSPAccount.DefaultAsset).To(Equal(expectedCurrency))
	Expect(resultingPSPAccount.Metadata).To(Equal(map[string]string{
		"beneficiary_id":                     beneficiary.Id,
		"bank_account_number":                beneficiary.BankAccount.AccountNUmber,
		"bank_account_iban":                  beneficiary.BankAccount.Iban,
		"bank_account_bic":                   beneficiary.BankAccount.Bic,
		"bank_account_swift_sort_code":       beneficiary.BankAccount.SwiftSortCode,
		"bank_account_routing_number":        beneficiary.BankAccount.RoutingNumber,
		"bank_account_intermediary_bank_bic": beneficiary.BankAccount.IntermediaryBankBic,
		"updated_at":                         beneficiary.UpdatedAt,
	}))
	Expect(resultingPSPAccount.Raw).To(Equal(expectedRaw))
}
