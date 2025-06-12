package qonto

import (
	"encoding/json"
	"github.com/formancehq/go-libs/v3/pointer"
	"math/big"
	"strconv"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/qonto/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Qonto *Plugin Balances", func() {
	Context("fetch next balances", func() {
		var (
			plg *Plugin
			m   *client.MockClient
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{client: m}
		})

		It("should fetch next balance", func(ctx SpecContext) {
			// Given
			fixturePSPAccount := getFixturePSPAccount()
			marshalledPSPAccount, _ := json.Marshal(fixturePSPAccount)
			req := models.FetchNextBalancesRequest{
				FromPayload: marshalledPSPAccount,
			}

			// When
			resp, err := plg.FetchNextBalances(ctx, req)

			// Then
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(1))
			balance := resp.Balances[0]
			Expect(balance.CreatedAt).To(Not(Equal(fixturePSPAccount.CreatedAt))) // implementation is time.now()
			Expect(balance.Asset).To(Equal(*fixturePSPAccount.DefaultAsset))
			Expect(balance.Amount).To(Equal(big.NewInt(12345678900)))
			Expect(balance.AccountReference).To(Equal(fixturePSPAccount.Reference))
		})

		Describe("Error cases", func() {
			It("missing fromPayload in request", func(ctx SpecContext) {
				// Given
				req := models.FetchNextBalancesRequest{}

				// When
				resp, err := plg.FetchNextBalances(ctx, req)

				// Then
				Expect(err).ToNot(BeNil())
				Expect(err).To(MatchError("missing from payload in request"))
				Expect(resp).To(Equal(models.FetchNextBalancesResponse{}))
			})

			It("invalid fromPayload in request", func(ctx SpecContext) {
				// Given
				req := models.FetchNextBalancesRequest{
					FromPayload: []byte(`{invalid: "PSPAccount"}`),
				}

				// When
				resp, err := plg.FetchNextBalances(ctx, req)

				// Then
				Expect(err).ToNot(BeNil())
				Expect(err).To(MatchError(ContainSubstring("failed to unmarshall FromPayload")))
				Expect(resp).To(Equal(models.FetchNextBalancesResponse{}))
			})

			It("missing fromPayload.DefaultAsset in request", func(ctx SpecContext) {
				// Given
				fixturePSPAccount := getFixturePSPAccount()
				fixturePSPAccount.DefaultAsset = nil
				marshalledPSPAccount, _ := json.Marshal(fixturePSPAccount)
				req := models.FetchNextBalancesRequest{
					FromPayload: marshalledPSPAccount,
				}

				// When
				resp, err := plg.FetchNextBalances(ctx, req)

				// Then
				Expect(err).ToNot(BeNil())
				Expect(err).To(MatchError(ContainSubstring("missing default asset")))
				Expect(resp).To(Equal(models.FetchNextBalancesResponse{}))
			})
		})
	})
})

func getFixturePSPAccount() models.PSPAccount {
	sampleQontoAccount := client.OrganizationBankAccount{
		Id:                     "1",
		Slug:                   "slug",
		Iban:                   "FR7630006000011234567890189",
		Bic:                    "BNPAFRPP",
		Currency:               "EUR",
		Balance:                "123456789",
		BalanceCents:           12345678900,
		AuthorizedBalance:      "123457789",
		AuthorizedBalanceCents: 12345778900,
		Name:                   "Sample",
		UpdatedAt:              "2021-01-01T00:00:00.001Z",
		Status:                 "active",
		Main:                   false,
		IsExternalAccount:      false,
		AccountNumber:          "1",
	}
	marshalledQontoAccount, _ := json.Marshal(sampleQontoAccount)
	createAt, _ := time.ParseInLocation(client.QontoTimeformat, sampleQontoAccount.UpdatedAt, time.UTC)
	return models.PSPAccount{
		Reference:    "1",
		CreatedAt:    createAt,
		Name:         &sampleQontoAccount.Name,
		DefaultAsset: pointer.For("EUR/2"),
		Metadata: map[string]string{
			"iban":                sampleQontoAccount.Iban,
			"bic":                 sampleQontoAccount.Bic,
			"account_number":      sampleQontoAccount.AccountNumber,
			"status":              sampleQontoAccount.Status,
			"is_external_account": strconv.FormatBool(sampleQontoAccount.IsExternalAccount),
			"main":                strconv.FormatBool(sampleQontoAccount.Main),
		},
		Raw: marshalledQontoAccount,
	}
}
