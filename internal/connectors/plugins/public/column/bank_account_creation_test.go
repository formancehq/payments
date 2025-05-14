package column

import (
	"errors"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/column/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Column Plugin Bank Accounts", func() {
	var (
		ctrl           *gomock.Controller
		mockHTTPClient *client.MockHTTPClient
		plg            models.Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockHTTPClient = client.NewMockHTTPClient(ctrl)
		c := client.New("test", "aseplye", "https://test.com")
		c.SetHttpClient(mockHTTPClient)
		plg = &Plugin{client: c}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Create counterparty Bank Accounts", func() {
		It("should return an error - required account number", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{
				BankAccount: models.BankAccount{},
			}

			_, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(ErrAccountNumberRequired.Error()))

		})

		It("should return an error - required routing number", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{
				BankAccount: models.BankAccount{
					AccountNumber: pointer.For("test_account_number"),
				},
			}

			_, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(ErrMissingRoutingNumber.Error()))

		})

		It("should return an error - required city when address is provided", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{
				BankAccount: models.BankAccount{
					AccountNumber: pointer.For("test_account_number"),
					Metadata: map[string]string{
						client.ColumnRoutingNumberMetadataKey: "test_routing_number",
						client.ColumnAddressLine1MetadataKey:  "123 Main St",
					},
				},
			}

			_, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(ErrMissingMetadataAddressCity.Error()))

		})

		It("should return an error - required country when address is provided", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{
				BankAccount: models.BankAccount{
					AccountNumber: pointer.For("test_account_number"),
					Metadata: map[string]string{
						client.ColumnRoutingNumberMetadataKey: "test_routing_number",
						client.ColumnAddressLine1MetadataKey:  "123 Main St",
						client.ColumnAddressCityMetadataKey:   "New York",
					},
				},
			}

			_, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(ErrMissingCountry.Error()))

		})

		It("should return error when city is provided without address line 1", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{
				BankAccount: models.BankAccount{
					AccountNumber: pointer.For("test_account_number"),
					Metadata: map[string]string{
						client.ColumnRoutingNumberMetadataKey: "test_routing_number",
						client.ColumnAddressCityMetadataKey:   "New York",
					},
				},
			}

			_, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(ErrMetadataAddressCityNotRequired.Error()))
		})

		It("should return error when addressLine2 is provided without address line 1", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{
				BankAccount: models.BankAccount{
					AccountNumber: pointer.For("test_account_number"),
					Metadata: map[string]string{
						client.ColumnRoutingNumberMetadataKey: "test_routing_number",
						client.ColumnAddressLine2MetadataKey:  "address line 2",
					},
				},
			}

			_, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(ErrMetadataAddressLine2NotRequired.Error()))

		})

		It("should return error when state is provided without address line 1", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{
				BankAccount: models.BankAccount{
					AccountNumber: pointer.For("test_account_number"),
					Metadata: map[string]string{
						client.ColumnRoutingNumberMetadataKey: "test_routing_number",
						client.ColumnAddressStateMetadataKey:  "NY",
					},
				},
			}

			_, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(ErrMetadataAddressStateNotRequired.Error()))

		})

		It("should return error when postal code is provided without address line 1", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{
				BankAccount: models.BankAccount{
					AccountNumber: pointer.For("test_account_number"),
					Metadata: map[string]string{
						client.ColumnRoutingNumberMetadataKey:     "test_routing_number",
						client.ColumnAddressPostalCodeMetadataKey: "10001",
					},
				},
			}

			_, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(ErrMetadataPostalCodeNotRequired.Error()))

		})

		It("should return error when country is provided without address line 1", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{
				BankAccount: models.BankAccount{
					AccountNumber: pointer.For("test_account_number"),
					Metadata: map[string]string{
						client.ColumnRoutingNumberMetadataKey: "test_routing_number",
					},
					Country: pointer.For("US"),
				},
			}

			_, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(ErrCountryNotRequired.Error()))

		})

		It("should return an error when HTTP request fails", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{
				BankAccount: models.BankAccount{
					AccountNumber: pointer.For("test_account_number"),
					Country:       pointer.For("US"),
					Metadata: map[string]string{
						client.ColumnRoutingNumberMetadataKey:      "test_routing_number",
						client.ColumnAddressLine1MetadataKey:       "123 Main St",
						client.ColumnAddressCityMetadataKey:        "New York",
						client.ColumnAddressCountryCodeMetadataKey: "US",
					},
				},
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

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
			Expect(err.Error()).To(ContainSubstring("test error"))
		})

		It("should return an error when parsing creation timestamp fails", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{
				BankAccount: models.BankAccount{
					AccountNumber: pointer.For("test_account_number"),
					Country:       pointer.For("US"),
					Metadata: map[string]string{
						client.ColumnRoutingNumberMetadataKey:      "test_routing_number",
						client.ColumnAccountNumberMetadataKey:      "test_account_number",
						client.ColumnAddressLine1MetadataKey:       "123 Main St",
						client.ColumnAddressCityMetadataKey:        "New York",
						client.ColumnAddressCountryCodeMetadataKey: "US",
					},
				},
			}

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.CounterPartyBankAccountResponse{
				ID:        "test_id",
				CreatedAt: "invalid-timestamp",
			})

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
			Expect(err.Error()).To(ContainSubstring("failed to parse creation time"))
		})

		It("should successfully create a bank account", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{
				BankAccount: models.BankAccount{
					AccountNumber: pointer.For("test_account_number"),
					Country:       pointer.For("US"),
					Metadata: map[string]string{
						client.ColumnRoutingNumberMetadataKey:      "test_routing_number",
						client.ColumnAddressLine1MetadataKey:       "123 Main St",
						client.ColumnAddressCityMetadataKey:        "New York",
						client.ColumnAddressCountryCodeMetadataKey: "US",
					},
				},
			}

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.CounterPartyBankAccountResponse{
				ID:        "test_id",
				Name:      "Test Account",
				CreatedAt: "2024-03-04T10:00:00Z",
				Address: client.Address{
					City:        "New York",
					CountryCode: "US",
					Line1:       "123 Main St",
					PostalCode:  "10001",
					State:       "NY",
				},
				AccountNumber: "test_account_number",
				RoutingNumber: "test_routing_number",
			})

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).To(BeNil())
			Expect(res).ToNot(Equal(models.CreateBankAccountResponse{}))
			Expect(res.RelatedAccount.Reference).To(Equal("test_id"))
			Expect(res.RelatedAccount.Name).To(Equal(pointer.For("Test Account")))
			Expect(res.RelatedAccount.CreatedAt).ToNot(BeZero())
			Expect(res.RelatedAccount.Raw).ToNot(BeNil())
		})
	})

})
