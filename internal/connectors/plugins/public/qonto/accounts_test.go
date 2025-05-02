package qonto

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/qonto/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Qonto *Plugin Accounts", func() {
	Context("fetch next accounts", func() {
		var (
			plg                  *Plugin
			m                    *client.MockClient
			pageSize             int
			sampleAccounts       []client.OrganizationBankAccount
			sortedSampleAccounts []client.OrganizationBankAccount
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{client: m}
			pageSize = 50

			sampleAccounts, sortedSampleAccounts = generateTestSampleAccounts()
		})

		It("should return an error - get organization error", func(ctx SpecContext) {
			// Given
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: pageSize,
			}

			m.EXPECT().GetOrganization(gomock.Any()).Return(
				nil,
				errors.New("test error"),
			)

			// When
			resp, err := plg.FetchNextAccounts(ctx, req)

			// Then
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextAccountsResponse{}))
		})

		It("should return an error - missing pageSize in request", func(ctx SpecContext) {
			// Given
			req := models.FetchNextAccountsRequest{
				State: []byte(`{}`),
			}

			m.EXPECT().GetOrganization(gomock.Any()).AnyTimes().Return(
				&client.Organization{},
				nil,
			)

			// When
			resp, err := plg.FetchNextAccounts(ctx, req)

			// Then
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("invalid request, missing page size in request"))
			Expect(resp).To(Equal(models.FetchNextAccountsResponse{}))
		})

		It("should fetch next accounts - no state no results", func(ctx SpecContext) {
			// Given
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: pageSize,
			}

			m.EXPECT().GetOrganization(gomock.Any()).Return(
				&client.Organization{},
				nil,
			)

			// When
			resp, err := plg.FetchNextAccounts(ctx, req)

			// Then
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(0))

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastUpdatedAt).To(Equal(time.Time{}))
		})

		It("should fetch next accounts - nil state", func(ctx SpecContext) {
			// Given
			req := models.FetchNextAccountsRequest{
				State:    nil,
				PageSize: pageSize,
			}

			m.EXPECT().GetOrganization(gomock.Any()).Return(
				&client.Organization{},
				nil,
			)

			// When
			resp, err := plg.FetchNextAccounts(ctx, req)

			// Then
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(0))

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastUpdatedAt).To(Equal(time.Time{}))
		})

		It("should fetch next accounts - no state", func(ctx SpecContext) {
			// Given
			req := models.FetchNextAccountsRequest{
				State:    []byte(`{}`),
				PageSize: pageSize,
			}

			m.EXPECT().GetOrganization(gomock.Any()).Return(
				&client.Organization{
					BankAccounts: sampleAccounts,
				},
				nil,
			)

			// When
			resp, err := plg.FetchNextAccounts(ctx, req)

			// Then
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(20))
			for i, account := range resp.Accounts {
				assertAccountMapping(sortedSampleAccounts[i], account)
			}

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastUpdatedAt.Format(client.QONTO_TIMEFORMAT)).To(Equal(sortedSampleAccounts[19].UpdatedAt))
		})

		It("should fetch next accounts - state set", func(ctx SpecContext) {
			// Given
			req := models.FetchNextAccountsRequest{
				State:    []byte(fmt.Sprintf(`{"lastUpdatedAt": "%v"}`, sortedSampleAccounts[9].UpdatedAt)),
				PageSize: pageSize,
			}
			m.EXPECT().GetOrganization(gomock.Any()).Return(
				&client.Organization{
					BankAccounts: sampleAccounts,
				},
				nil,
			)

			// When
			resp, err := plg.FetchNextAccounts(ctx, req)

			// Then
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(10))
			for i, account := range resp.Accounts {
				assertAccountMapping(sortedSampleAccounts[i+10], account)
			}

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastUpdatedAt.Format(client.QONTO_TIMEFORMAT)).To(Equal(sortedSampleAccounts[19].UpdatedAt))
			Expect(resp.HasMore).To(BeFalse())
		})

		It("should fetch next accounts - no state and pageSize < total", func(ctx SpecContext) {
			// Given
			req := models.FetchNextAccountsRequest{
				State:    []byte("{}"),
				PageSize: 5,
			}
			m.EXPECT().GetOrganization(gomock.Any()).Return(
				&client.Organization{
					BankAccounts: sampleAccounts,
				},
				nil,
			)

			// When
			resp, err := plg.FetchNextAccounts(ctx, req)

			// Then
			Expect(err).To(BeNil())
			Expect(resp.Accounts).To(HaveLen(5))
			for i, account := range resp.Accounts {
				assertAccountMapping(sortedSampleAccounts[i], account)
			}

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastUpdatedAt.Format(client.QONTO_TIMEFORMAT)).To(Equal(sortedSampleAccounts[4].UpdatedAt))
			Expect(resp.HasMore).To(BeTrue())
		})

	})
})

/**
 * Generate shuffled tests data, and keep a sorted copy for assertion (the call under test is modifying the data in place)
 */
func generateTestSampleAccounts() (sampleAccounts []client.OrganizationBankAccount, sortedSampleAccounts []client.OrganizationBankAccount) {
	sampleAccounts = make([]client.OrganizationBankAccount, 0)
	for i := 0; i < 20; i++ {
		main, isExternalAccount := i == 0, i == 1

		sampleAccounts = append(sampleAccounts, client.OrganizationBankAccount{
			Id:                fmt.Sprintf("%d", i),
			Name:              fmt.Sprintf("Account %d", i),
			Iban:              fmt.Sprintf("FR%02d0000000%02d", i, i),
			Currency:          "EUR",
			Balance:           float64(i),
			BalanceCents:      int64(i) * 100,
			Status:            "active",
			UpdatedAt:         fmt.Sprintf("2021-01-01T00:%02d:00.001Z", i),
			Main:              main,
			IsExternalAccount: isExternalAccount,
		})
	}

	sortedSampleAccounts = make([]client.OrganizationBankAccount, len(sampleAccounts))
	copy(sortedSampleAccounts, sampleAccounts)

	// Shuffle the array to be like the api response.
	rand.Shuffle(len(sampleAccounts), func(i, j int) {
		sampleAccounts[i], sampleAccounts[j] = sampleAccounts[j], sampleAccounts[i]
	})
	return
}

func assertAccountMapping(sampleQontoAccount client.OrganizationBankAccount, resultingPSPAccount models.PSPAccount) {
	Expect(resultingPSPAccount.Reference).To(Equal(sampleQontoAccount.Id))
	Expect(*resultingPSPAccount.Name).To(Equal(sampleQontoAccount.Name))
	Expect(resultingPSPAccount.CreatedAt.Format(client.QONTO_TIMEFORMAT)).To(Equal(sampleQontoAccount.UpdatedAt))
	Expect(*resultingPSPAccount.DefaultAsset).To(Equal("EUR/2"))
	Expect(resultingPSPAccount.Metadata).To(Equal(map[string]string{
		"iban":                sampleQontoAccount.Iban,
		"bic":                 sampleQontoAccount.Bic,
		"account_number":      sampleQontoAccount.AccountNumber,
		"status":              sampleQontoAccount.Status,
		"is_external_account": strconv.FormatBool(sampleQontoAccount.IsExternalAccount),
		"main":                strconv.FormatBool(sampleQontoAccount.Main),
	}))
}
