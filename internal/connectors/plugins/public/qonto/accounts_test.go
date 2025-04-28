package qonto

import (
	"encoding/json"
	"errors"
	"fmt"
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
			plg *Plugin
			m   *client.MockClient

			sampleAccounts []client.QontoBankAccount
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{client: m}

			sampleAccounts = generateTestSampleAccounts()
		})

		It("should return an error - get accounts error", func(ctx SpecContext) {
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
			Expect(resp.Accounts).To(HaveLen(0)) // TODO actually check the mapping

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
				assertAccountMapping(sampleAccounts[i], account)
			}

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastUpdatedAt.Format("2006-01-02T15:04:05.999Z")).To(Equal(sampleAccounts[19].UpdatedAt))
		})

		It("should fetch next accounts - state set", func(ctx SpecContext) {
			// Given
			req := models.FetchNextAccountsRequest{
				State: []byte(fmt.Sprintf(`{"lastUpdatedAt": "%v"}`, sampleAccounts[9].UpdatedAt)),
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
				assertAccountMapping(sampleAccounts[i+10], account)
			}

			var state accountsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastUpdatedAt.Format("2006-01-02T15:04:05.999Z")).To(Equal(sampleAccounts[19].UpdatedAt))
		})

	})
})

func generateTestSampleAccounts() []client.QontoBankAccount {
	sampleAccounts := make([]client.QontoBankAccount, 0)
	for i := 0; i < 20; i++ {
		sampleAccounts = append(sampleAccounts, client.QontoBankAccount{
			Id:        fmt.Sprintf("%d", i),
			Name:      fmt.Sprintf("Account %d", i),
			Iban:      fmt.Sprintf("FR%02d0000000%02d", i, i),
			Currency:  "EUR",
			Balance:   float64(i) * 100.0,
			Status:    "active",
			UpdatedAt: fmt.Sprintf("2021-01-01T00:%02d:00.001Z", i),
		})
	}
	return sampleAccounts
}

func assertAccountMapping(sampleQontoAccount client.QontoBankAccount, resultingPSPAccount models.PSPAccount) {
	Expect(resultingPSPAccount.Reference).To(Equal(sampleQontoAccount.Id))
	Expect(*resultingPSPAccount.Name).To(Equal(sampleQontoAccount.Name))
	Expect(resultingPSPAccount.CreatedAt.Format("2006-01-02T15:04:05.999Z")).To(Equal(sampleQontoAccount.UpdatedAt))
	Expect(*resultingPSPAccount.DefaultAsset).To(Equal("EUR/2"))
	Expect(resultingPSPAccount.Metadata).To(BeEmpty())
}
