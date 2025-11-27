package increase

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
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
			mockHTTPClient    *client.MockHTTPClient
			sampleBankAccount models.BankAccount
			now               time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			mockHTTPClient = client.NewMockHTTPClient(ctrl)
			plg.client = client.New("test", "aseplye", "https://test.com", "we5432345")
			plg.client.SetHttpClient(mockHTTPClient)
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

		It("should return an error - create bank account error", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{
				BankAccount: sampleBankAccount,
			}

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				500,
				errors.New("test error"),
			)

			resp, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to create bank account: test error : : status code: 0"))
			Expect(resp).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - missing routingNumber in bank account metadata", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.Metadata[client.IncreaseRoutingNumberMetadataKey] = ""
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("validation error occurred for field com.increase.spec/routingNumber: missing required metadata in request"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - missing accountHolder in bank account metadata", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.Metadata[client.IncreaseAccountHolderMetadataKey] = ""
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("validation error occurred for field com.increase.spec/accountHolder: missing required metadata in request"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - missing description in bank account metadata", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.Metadata[client.IncreaseDescriptionMetadataKey] = ""
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("validation error occurred for field com.increase.spec/description: missing required metadata in request"))
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
				CreatedAt:     now.Format(time.RFC3339),
			}

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, *expectedBA)

			raw, _ := json.Marshal(expectedBA)

			createdAt, _ := time.Parse(time.RFC3339, expectedBA.CreatedAt)
			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(models.CreateBankAccountResponse{
				RelatedAccount: models.PSPAccount{
					Reference: "id",
					Name:      pointer.For("description"),
					CreatedAt: createdAt,
					Metadata: map[string]string{
						client.IncreaseAccountHolderMetadataKey: "business",
						client.IncreaseAccountNumberMetadataKey: "12345678",
						client.IncreaseDescriptionMetadataKey:   "description",
						client.IncreaseRoutingNumberMetadataKey: "23567655",
						client.IncreaseTypeMetadataKey:          "",
						client.IncreaseStatusMetadataKey:        "active",
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
			Expect(err).To(MatchError("validation error occurred for field AccountNumber: missing required field in request"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})
	})
})
