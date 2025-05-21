package gocardless

import (
	"errors"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/gocardless/client"
	"github.com/formancehq/payments/internal/models"
	gocardless "github.com/gocardless/gocardless-pro-go/v4"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Gocardless Plugin Bank Account Creation", func() {
	var (
		plg               *Plugin
		sampleBankAccount models.BankAccount
		now               time.Time
	)

	BeforeEach(func() {
		plg = &Plugin{}

	})

	Context("create bank account", func() {
		var (
			mockedService *client.MockGoCardlessService
		)

		BeforeEach((func() {
			ctrl := gomock.NewController(GinkgoT())
			mockedService = client.NewMockGoCardlessService(ctrl)

			plg.client, _ = client.New("test", "https://example.com", "access_token", true)
			plg.client.NewWithService(mockedService)
			now = time.Now().UTC()

			sampleBankAccount = models.BankAccount{
				ID:            uuid.New(),
				CreatedAt:     now.UTC(),
				Name:          "test",
				AccountNumber: pointer.For("20548790"),
				SwiftBicCode:  pointer.For("BNPAFRPP"),
				Country:       pointer.For("US"),
				Metadata: map[string]string{
					client.GocardlessCurrencyMetadataKey:    "USD",
					client.GocardlessAccountTypeMetadataKey: "savings",
				},
			}
		}))

		It("should return an error - required metadata field currency is missing", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.Metadata = map[string]string{}
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(ErrMissingCurrency))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - currency not supported", func(ctx SpecContext) {
			ba := sampleBankAccount

			ba.Metadata = map[string]string{
				client.GocardlessCurrencyMetadataKey:    "XYZ",
				client.GocardlessAccountTypeMetadataKey: "savings",
			}
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)

			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(ErrNotSupportedCurrency))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - com.gocardless.spec/customer ID format invalid", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.Metadata = map[string]string{
				client.GocardlessCurrencyMetadataKey:    "USD",
				client.GocardlessCustomerMetadataKey:    "INVALID123",
				client.GocardlessAccountTypeMetadataKey: "savings",
			}
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(ErrInvalidCustomerID))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - com.gocardless.spec/creditor ID format invalid", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.Metadata = map[string]string{
				client.GocardlessCurrencyMetadataKey:    "USD",
				client.GocardlessCreditorMetadataKey:    "INVALID123",
				client.GocardlessAccountTypeMetadataKey: "savings",
			}
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(ErrInvalidCreditorID))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - missing both com.gocardless.spec/customer and com.gocardless.spec/creditor", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.Metadata = map[string]string{
				client.GocardlessCurrencyMetadataKey:    "USD",
				client.GocardlessAccountTypeMetadataKey: "savings",
			}
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(ErrCreditorAndCustomerIDProvided))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - you must provide either com.gocardless.spec/customer or com.gocardless.spec/creditor metadata field but not both", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.Metadata = map[string]string{
				client.GocardlessCurrencyMetadataKey:    "USD",
				client.GocardlessCustomerMetadataKey:    "CU987654321",
				client.GocardlessCreditorMetadataKey:    "CR123456789",
				client.GocardlessAccountTypeMetadataKey: "savings",
			}
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
			Expect(err).To(MatchError(ErrCreditorAndCustomerIDProvided))
		})

		It("should return an error - required metadata field com.gocardless.spec/account_type is missing for US accounts", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.Country = pointer.For("US")
			ba.Metadata = map[string]string{
				client.GocardlessCurrencyMetadataKey: "USD",
				client.GocardlessCustomerMetadataKey: "CU123456789",
			}
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(ErrMissingAccountType))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - invalid com.gocardless.spec/account_type for US accounts", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.Country = pointer.For("US")
			ba.Metadata = map[string]string{
				client.GocardlessCurrencyMetadataKey:    "USD",
				client.GocardlessCustomerMetadataKey:    "CU123456789",
				client.GocardlessAccountTypeMetadataKey: "invalid",
			}
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(ErrInvalidAccountType))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - when account_type is provided for non USD account", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.Country = pointer.For("US")
			ba.Metadata = map[string]string{
				client.GocardlessCurrencyMetadataKey:    "GBP",
				client.GocardlessCustomerMetadataKey:    "CU123456789",
				client.GocardlessAccountTypeMetadataKey: "savings",
			}
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(ErrAccountTypeProvided))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - account number is required", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.AccountNumber = nil
			ba.Metadata = map[string]string{
				client.GocardlessCurrencyMetadataKey:    "USD",
				client.GocardlessCreditorMetadataKey:    "CR123",
				client.GocardlessAccountTypeMetadataKey: "savings",
			}
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(ErrMissingAccountNumber))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - swift bic code is required", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.SwiftBicCode = nil
			ba.Metadata = map[string]string{
				client.GocardlessCurrencyMetadataKey:    "USD",
				client.GocardlessCreditorMetadataKey:    "CR123",
				client.GocardlessAccountTypeMetadataKey: "savings",
			}
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(ErrMissingSwiftCode))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - country is required", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.Country = nil
			ba.Metadata = map[string]string{
				client.GocardlessCurrencyMetadataKey:    "USD",
				client.GocardlessCreditorMetadataKey:    "CR123",
				client.GocardlessAccountTypeMetadataKey: "savings",
			}
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(ErrorMissingCountry))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		DescribeTable("create bank accounts successfully",
			func(ctx SpecContext, accountType string, metadata map[string]string, expectedLinks interface{}) {
				ba := sampleBankAccount
				ba.Metadata = metadata

				var expectedAccount interface{}
				if accountType == "creditor" {
					expectedAccount = &gocardless.CreditorBankAccount{
						Id:                "ba_123",
						CreatedAt:         now.Format(time.RFC3339),
						AccountHolderName: ba.Name,
						Currency:          "USD",
						Metadata:          map[string]interface{}{},
						CountryCode:       "US",
						AccountType:       "savings",
						Links:             expectedLinks.(*gocardless.CreditorBankAccountLinks),
					}
					mockedService.EXPECT().CreateGocardlessCreditorBankAccount(gomock.Any(), gocardless.CreditorBankAccountCreateParams{
						AccountHolderName: ba.Name,
						AccountNumber:     *ba.AccountNumber,
						AccountType:       ba.Metadata[client.GocardlessAccountTypeMetadataKey],
						BankCode:          *ba.SwiftBicCode,
						CountryCode:       *ba.Country,
						Currency:          ba.Metadata[client.GocardlessCurrencyMetadataKey],
						Links:             gocardless.CreditorBankAccountCreateParamsLinks{Creditor: ba.Metadata[client.GocardlessCreditorMetadataKey]},
					}).Return(expectedAccount, nil)
				} else {
					expectedAccount = &gocardless.CustomerBankAccount{
						Id:                "ba_123",
						CreatedAt:         now.Format(time.RFC3339),
						AccountHolderName: ba.Name,
						Currency:          "USD",
						Metadata:          map[string]interface{}{},
						CountryCode:       "US",
						AccountType:       "savings",
						Links:             expectedLinks.(*gocardless.CustomerBankAccountLinks),
					}
					mockedService.EXPECT().CreateGocardlessCustomerBankAccount(gomock.Any(), gocardless.CustomerBankAccountCreateParams{
						AccountHolderName: ba.Name,
						AccountNumber:     *ba.AccountNumber,
						AccountType:       ba.Metadata[client.GocardlessAccountTypeMetadataKey],
						BankCode:          *ba.SwiftBicCode,
						CountryCode:       *ba.Country,
						Currency:          ba.Metadata[client.GocardlessCurrencyMetadataKey],
						Links:             gocardless.CustomerBankAccountCreateParamsLinks{Customer: ba.Metadata[client.GocardlessCustomerMetadataKey]},
					}).Return(expectedAccount, nil)
				}

				res, err := plg.createBankAccount(ctx, ba)
				Expect(err).To(BeNil())
				Expect(res.RelatedAccount.Reference).To(Equal("ba_123"))
				Expect(*res.RelatedAccount.Name).To(Equal(ba.Name))
				Expect(res.RelatedAccount.Metadata).To(Equal(map[string]string{
					client.GocardlessAccountTypeMetadataKey: "savings",
				}))
				Expect(*res.RelatedAccount.DefaultAsset).To(Equal("USD/2"))
			},
			Entry("creditor bank account",
				"creditor",
				map[string]string{
					client.GocardlessCurrencyMetadataKey:    "USD",
					client.GocardlessCreditorMetadataKey:    "CR123",
					client.GocardlessAccountTypeMetadataKey: "savings",
				},
				&gocardless.CreditorBankAccountLinks{
					Creditor: "CR123",
				},
			),
			Entry("customer bank account",
				"customer",
				map[string]string{
					client.GocardlessCurrencyMetadataKey:    "USD",
					client.GocardlessCustomerMetadataKey:    "CU123",
					client.GocardlessAccountTypeMetadataKey: "savings",
				},
				&gocardless.CustomerBankAccountLinks{
					Customer: "CU123",
				},
			),
		)

		DescribeTable("should return error when client failed",
			func(ctx SpecContext, accountType string, metadata map[string]string, expectedError interface{}) {
				ba := sampleBankAccount
				ba.Metadata = metadata

				if accountType == "creditor" {
					mockedService.EXPECT().CreateGocardlessCreditorBankAccount(gomock.Any(), gomock.Any(),
						gomock.Any(),
					).Return(nil, expectedError)
				} else {
					mockedService.EXPECT().CreateGocardlessCustomerBankAccount(gomock.Any(), gomock.Any(),
						gomock.Any(),
					).Return(nil, expectedError)
				}

				resp, err := plg.createBankAccount(ctx, ba)
				Expect(err).NotTo(BeNil())
				Expect(resp).To(Equal(models.CreateBankAccountResponse{}))
				Expect(err).To(MatchError("create account error"))
			},
			Entry("creditor bank account",
				"creditor",
				map[string]string{
					client.GocardlessCurrencyMetadataKey:    "USD",
					client.GocardlessCreditorMetadataKey:    "CR123",
					client.GocardlessAccountTypeMetadataKey: "savings",
				},
				errors.New("create account error"),
			),
			Entry("customer bank account",
				"customer",
				map[string]string{
					client.GocardlessCurrencyMetadataKey:    "USD",
					client.GocardlessCustomerMetadataKey:    "CU123",
					client.GocardlessAccountTypeMetadataKey: "savings",
				},
				errors.New("create account error"),
			),
		)

		DescribeTable("should return error when gocardless returns an error",
			func(ctx SpecContext, accountType string, metadata map[string]string, expectedError interface{}) {
				ba := sampleBankAccount
				ba.Metadata = metadata

				if accountType == "creditor" {
					mockedService.EXPECT().CreateGocardlessCreditorBankAccount(gomock.Any(), gomock.Any(),
						gomock.Any(),
					).Return(nil, expectedError)
				} else {
					mockedService.EXPECT().CreateGocardlessCustomerBankAccount(gomock.Any(), gomock.Any(),
						gomock.Any(),
					).Return(nil, expectedError)
				}

				resp, err := plg.createBankAccount(ctx, ba)
				Expect(err).NotTo(BeNil())
				Expect(resp).To(Equal(models.CreateBankAccountResponse{}))
				Expect(err).To(MatchError("create account error"))
			},
			Entry("creditor bank account",
				"creditor",
				map[string]string{
					client.GocardlessCurrencyMetadataKey:    "USD",
					client.GocardlessCreditorMetadataKey:    "CR123",
					client.GocardlessAccountTypeMetadataKey: "savings",
				},
				errors.New("create account error"),
			),
			Entry("customer bank account",
				"customer",
				map[string]string{
					client.GocardlessCurrencyMetadataKey:    "USD",
					client.GocardlessCustomerMetadataKey:    "CU123",
					client.GocardlessAccountTypeMetadataKey: "savings",
				},
				errors.New("create account error"),
			),
		)

		DescribeTable("should return error when timestamp is invalid",
			func(ctx SpecContext, accountType string, metadata map[string]string, expectedLinks interface{}) {
				ba := sampleBankAccount
				ba.Metadata = metadata

				var expectedAccount interface{}
				if accountType == "creditor" {
					expectedAccount = &gocardless.CreditorBankAccount{
						CreatedAt: "invalid",
					}
					mockedService.EXPECT().CreateGocardlessCreditorBankAccount(gomock.Any(), gomock.Any()).Return(expectedAccount, nil)
				} else {
					expectedAccount = &gocardless.CustomerBankAccount{
						CreatedAt: "invalid",
					}
					mockedService.EXPECT().CreateGocardlessCustomerBankAccount(gomock.Any(), gomock.Any(),
						gomock.Any(),
					).Return(expectedAccount, nil)
				}

				res, err := plg.createBankAccount(ctx, ba)
				Expect(err).ToNot(BeNil())
				Expect(res).To(Equal(models.CreateBankAccountResponse{}))
				Expect(err.Error()).To(ContainSubstring("failed to parse creation time"))
			},
			Entry("creditor bank account",
				"creditor",
				map[string]string{
					client.GocardlessCurrencyMetadataKey:    "USD",
					client.GocardlessCreditorMetadataKey:    "CR123",
					client.GocardlessAccountTypeMetadataKey: "savings",
				},
				&gocardless.CreditorBankAccountLinks{
					Creditor: "CR123",
				},
			),
			Entry("customer bank account",
				"customer",
				map[string]string{
					client.GocardlessCurrencyMetadataKey:    "USD",
					client.GocardlessCustomerMetadataKey:    "CU123",
					client.GocardlessAccountTypeMetadataKey: "savings",
				},
				&gocardless.CustomerBankAccountLinks{
					Customer: "CU123",
				},
			),
		)
	})

	Context("create bank account failed", func() {
		var (
			m *client.MockClient
		)

		BeforeEach((func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)

			plg.client = m
			now = time.Now().UTC()

			sampleBankAccount = models.BankAccount{
				ID:            uuid.New(),
				CreatedAt:     now.UTC(),
				Name:          "test",
				AccountNumber: pointer.For("20548790"),
				SwiftBicCode:  pointer.For("BNPAFRPP"),
				Country:       pointer.For("US"),
				Metadata: map[string]string{
					client.GocardlessCurrencyMetadataKey:    "USD",
					client.GocardlessAccountTypeMetadataKey: "savings",
				},
			}

		}))

		DescribeTable("should return error when gocardless returns an error",
			func(ctx SpecContext, accountType string, metadata map[string]string, expectedError interface{}) {
				ba := sampleBankAccount
				ba.Metadata = metadata

				if accountType == "creditor" {
					m.EXPECT().CreateCreditorBankAccount(gomock.Any(), gomock.Any(),
						gomock.Any(),
					).Return(client.GocardlessGenericAccount{}, expectedError)
				} else {
					m.EXPECT().CreateCustomerBankAccount(gomock.Any(), gomock.Any(),
						gomock.Any(),
					).Return(client.GocardlessGenericAccount{}, expectedError)
				}

				resp, err := plg.createBankAccount(ctx, ba)
				Expect(err).NotTo(BeNil())
				Expect(resp).To(Equal(models.CreateBankAccountResponse{}))
				Expect(err).To(MatchError("create account error"))
			},
			Entry("creditor bank account",
				"creditor",
				map[string]string{
					client.GocardlessCurrencyMetadataKey:    "USD",
					client.GocardlessCreditorMetadataKey:    "CR123",
					client.GocardlessAccountTypeMetadataKey: "savings",
				},
				errors.New("create account error"),
			),
			Entry("customer bank account",
				"customer",
				map[string]string{
					client.GocardlessCurrencyMetadataKey:    "USD",
					client.GocardlessCustomerMetadataKey:    "CU123",
					client.GocardlessAccountTypeMetadataKey: "savings",
				},
				errors.New("create account error"),
			),
		)

	})
})
