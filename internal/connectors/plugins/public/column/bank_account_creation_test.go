package column

import (
	"errors"
	"fmt"

	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/column/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Column Plugin Bank Accounts", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("Create counterparty Bank Accounts", func() {
		var (
			mockHTTPClient *client.MockHTTPClient
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			mockHTTPClient = client.NewMockHTTPClient(ctrl)
			plg.client = client.New("test", "aseplye", "https://test.com")
			plg.client.SetHttpClient(mockHTTPClient)
		})

		It("should return an error - required routing number", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{
				BankAccount: models.BankAccount{
					AccountNumber: pointer.For("test_account_number"),
				},
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("required metadata field %s is missing", client.ColumnRoutingNumberMetadataKey)))

		})

		It("should return an error - required city when address is provided", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{
				BankAccount: models.BankAccount{
					AccountNumber: pointer.For("test_account_number"),
					Metadata: map[string]string{
						client.ColumnRoutingNumberMetadataKey: "test_routing_number",

						client.ColumnAddressLine1MetadataKey: "123 Main St",
					},
				},
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("required metadata field %s is missing", client.ColumnCityMetadataKey)))
		})

		It("should return an error - required country when address is provided", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{
				BankAccount: models.BankAccount{
					AccountNumber: pointer.For("test_account_number"),
					Metadata: map[string]string{
						client.ColumnRoutingNumberMetadataKey: "test_routing_number",
						client.ColumnAddressLine1MetadataKey:  "123 Main St",
						client.ColumnCityMetadataKey:          "New York",
					},
				},
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
			Expect(err.Error()).To(ContainSubstring("country is required"))
		})

		It("should return error when city is provided without address line 1", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{
				BankAccount: models.BankAccount{
					AccountNumber: pointer.For("test_account_number"),
					Metadata: map[string]string{
						client.ColumnRoutingNumberMetadataKey: "test_routing_number",
						client.ColumnCityMetadataKey:          "New York",
					},
				},
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
			Expect(err).To(MatchError(fmt.Sprintf("metadata field %s is not required when addressLine1 is not provided", client.ColumnCityMetadataKey)))
		})

		It("should return error when state is provided without address line 1", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{
				BankAccount: models.BankAccount{
					AccountNumber: pointer.For("test_account_number"),
					Metadata: map[string]string{
						client.ColumnRoutingNumberMetadataKey: "test_routing_number",
						client.ColumnStateMetadataKey:         "NY",
					},
				},
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
			Expect(err).To(MatchError(fmt.Sprintf("metadata field %s is not required when addressLine1 is not provided", client.ColumnStateMetadataKey)))
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

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
			Expect(err).To(MatchError(fmt.Sprintf("metadata field %s is not required when addressLine1 is not provided", client.ColumnAddressPostalCodeMetadataKey)))
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

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
			Expect(err).To(MatchError("metadata field country is not required when addressLine1 is not provided"))
		})

		It("should return an error when HTTP request fails", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{
				BankAccount: models.BankAccount{
					AccountNumber: pointer.For("test_account_number"),
					Country:       pointer.For("US"),
					Metadata: map[string]string{
						client.ColumnRoutingNumberMetadataKey: "test_routing_number",
						client.ColumnAddressLine1MetadataKey:  "123 Main St",
						client.ColumnCityMetadataKey:          "New York",
						client.ColumnCountryCodeMetadataKey:   "US",
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

		It("should return an error when creating request fails", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{
				BankAccount: models.BankAccount{
					AccountNumber: pointer.For("test_account_number"),
					Country:       pointer.For("US"),
					Metadata: map[string]string{
						client.ColumnRoutingNumberMetadataKey: "test_routing_number",
						client.ColumnAddressLine1MetadataKey:  "123 Main St",
						client.ColumnCityMetadataKey:          "New York",
						client.ColumnCountryCodeMetadataKey:   "US",
					},
				},
			}

			plg.client = client.New("test", "aseplye", "http://invalid:port")
			plg.client.SetHttpClient(mockHTTPClient)

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
			Expect(err.Error()).To(ContainSubstring("failed to create counter party bank account"))
		})

		It("should return an error when parsing creation timestamp fails", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{
				BankAccount: models.BankAccount{
					AccountNumber: pointer.For("test_account_number"),
					Country:       pointer.For("US"),
					Metadata: map[string]string{
						client.ColumnRoutingNumberMetadataKey: "test_routing_number",
						client.ColumnAccountNumberMetadataKey: "test_account_number",
						client.ColumnAddressLine1MetadataKey:  "123 Main St",
						client.ColumnCityMetadataKey:          "New York",
						client.ColumnCountryCodeMetadataKey:   "US",
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
						client.ColumnRoutingNumberMetadataKey: "test_routing_number",
						client.ColumnAddressLine1MetadataKey:  "123 Main St",
						client.ColumnCityMetadataKey:          "New York",
						client.ColumnCountryCodeMetadataKey:   "US",
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
