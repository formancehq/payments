package increase

import (
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Increase Plugin Bank Account Creation", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("create bank account", func() {
		var (
			m                 *client.MockClient
			sampleBankAccount models.BankAccount
			now               time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
			now = time.Now().UTC()

			sampleBankAccount = models.BankAccount{
				ID:            uuid.New(),
				CreatedAt:     now.UTC(),
				Name:          "test",
				AccountNumber: pointer.For("12345678"),
				Metadata: map[string]string{
					client.IncreaseAccountHolderMetadataKey: "business",
					client.IncreaseDescriptionMetadataKey:   "description",
					client.IncreaseRoutingNumberMetadataKey: "23567655",
				},
			}
		})

		It("should return an error - missing routingNumber in bank account metadata", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.Metadata = map[string]string{}
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("missing routingNumber in bank account metadata: invalid request"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should be ok - create a bank account", func(ctx SpecContext) {
			ba := sampleBankAccount
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			expectedBA := &client.BankAccountResponse{
				ID:            "id",
				AccountNumber: *ba.AccountNumber,
				Description:   ba.Metadata[client.IncreaseDescriptionMetadataKey],
				RoutingNumber: ba.Metadata[client.IncreaseRoutingNumberMetadataKey],
				AccountHolder: ba.Metadata[client.IncreaseAccountHolderMetadataKey],
				Status:        "active",
			}
			m.EXPECT().CreateBankAccount(gomock.Any(), &client.BankAccountRequest{
				AccountNumber: *ba.AccountNumber,
				AccountHolder: ba.Metadata[client.IncreaseAccountHolderMetadataKey],
				RoutingNumber: ba.Metadata[client.IncreaseRoutingNumberMetadataKey],
				Description:   ba.Metadata[client.IncreaseDescriptionMetadataKey],
			}).Return(
				expectedBA,
				nil,
			)

			raw, _ := json.Marshal(expectedBA)

			createdAt, _ := time.Parse("2006-01-02T15:04:05.999-0700", expectedBA.CreatedAt)
			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(models.CreateBankAccountResponse{
				RelatedAccount: models.PSPAccount{
					Reference: "id",
					CreatedAt: createdAt,
					Metadata: map[string]string{
						"accountHolder": "business",
						"accountNumber": "12345678",
						"description":   "description",
						"routingNumber": "23567655",
						"type":          "",
						"status":        "active",
					},
					Raw: raw,
				},
			}))
		})

		It("should return an error - create bank account with missing account number", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.IBAN = nil
			ba.AccountNumber = nil
			ba.Country = pointer.For("TEST_COUNTRY")
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("missing accountNumber in bank account request: invalid request"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})
	})
})
