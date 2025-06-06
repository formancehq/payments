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
			sampleAccounts       []client.OrganizationBankAccount
			sortedSampleAccounts []client.OrganizationBankAccount
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{client: m}

			sampleAccounts, sortedSampleAccounts = generateTestSampleAccounts()
		})

		Describe("Error cases", func() {
			It("get organization error", func(ctx SpecContext) {
				// Given
				req := models.FetchNextAccountsRequest{
					State: []byte(`{}`),
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

			It("failing to unmarshall state", func(ctx SpecContext) {
				// Given
				req := models.FetchNextAccountsRequest{
					State: []byte(`{toto: "tata"}`),
				}

				m.EXPECT().GetOrganization(gomock.Any()).Times(0)

				// When
				resp, err := plg.FetchNextAccounts(ctx, req)

				// Then
				Expect(err).ToNot(BeNil())
				Expect(err).To(MatchError(ContainSubstring("failed to unmarshall state")))
				Expect(resp).To(Equal(models.FetchNextAccountsResponse{}))
			})

			It("invalid time format from Qonto's accounts", func(ctx SpecContext) {
				// Given an account with invalid timestamp
				req := models.FetchNextAccountsRequest{
					State: []byte(`{}`),
				}
				sampleAccounts[0].UpdatedAt = "202-01-01T00:00:00.001Z"

				m.EXPECT().GetOrganization(gomock.Any()).Return(
					&client.Organization{
						BankAccounts: sampleAccounts,
					},
					nil,
				)

				// When
				resp, err := plg.FetchNextAccounts(ctx, req)

				// Then
				Expect(err).ToNot(BeNil())
				Expect(err).To(MatchError(ContainSubstring("invalid time format for bank account")))
				Expect(resp).To(Equal(models.FetchNextAccountsResponse{}))
			})
		})

		It("should fetch next accounts - no state no results", func(ctx SpecContext) {
			// Given
			req := models.FetchNextAccountsRequest{
				State: []byte(`{}`),
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

		It("should fetch next accounts - nil state no results", func(ctx SpecContext) {
			// Given
			req := models.FetchNextAccountsRequest{
				State: nil,
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
				State: []byte(`{}`),
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
			Expect(state.LastUpdatedAt.Format(client.QontoTimeformat)).To(Equal(sortedSampleAccounts[19].UpdatedAt))
		})

		It("filters out already processed accounts based on lastUpdatedAt", func(ctx SpecContext) {
			// Given
			req := models.FetchNextAccountsRequest{
				State: []byte(fmt.Sprintf(`{"lastUpdatedAt": "%v"}`, sortedSampleAccounts[10].UpdatedAt)),
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
			Expect(state.LastUpdatedAt.Format(client.QontoTimeformat)).To(Equal(sortedSampleAccounts[19].UpdatedAt))
			Expect(resp.HasMore).To(BeFalse())
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
			Id:                string(rune('a' + i)),
			Name:              fmt.Sprintf("Account %d", i),
			Iban:              fmt.Sprintf("FR%02d0000000%02d", i, i),
			Currency:          "EUR",
			Balance:           json.Number(strconv.Itoa(i)),
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
	var expectedRaw json.RawMessage
	expectedRaw, _ = json.Marshal(sampleQontoAccount)
	Expect(resultingPSPAccount.Reference).To(Equal(sampleQontoAccount.Id))
	Expect(*resultingPSPAccount.Name).To(Equal(sampleQontoAccount.Name))
	Expect(resultingPSPAccount.CreatedAt.Format(client.QontoTimeformat)).To(Equal(sampleQontoAccount.UpdatedAt))
	Expect(*resultingPSPAccount.DefaultAsset).To(Equal("EUR/2"))
	Expect(resultingPSPAccount.Metadata).To(Equal(map[string]string{
		"bank_account_iban":   sampleQontoAccount.Iban,
		"bank_account_bic":    sampleQontoAccount.Bic,
		"bank_account_number": sampleQontoAccount.AccountNumber,
		"status":              sampleQontoAccount.Status,
		"is_external_account": strconv.FormatBool(sampleQontoAccount.IsExternalAccount),
		"main":                strconv.FormatBool(sampleQontoAccount.Main),
	}))
	Expect(resultingPSPAccount.Raw).To(Equal(expectedRaw))
}
