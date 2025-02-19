package gocardless

import (
	"time"

	"github.com/formancehq/go-libs/v2/pointer"
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
				IBAN:          pointer.For("FR7630006000011234567890189"),
				SwiftBicCode:  pointer.For("BNPAFRPP"),
				Country:       pointer.For("US"),
				Metadata: map[string]string{
					client.GocardlessBranchCodeMetadataKey:  "12345",
					client.GocardlessCurrencyMetadataKey:    "USD",
					client.GocardlessAccountTypeMetadataKey: "savings",
				},
			}

		}))

		It("should return an error - required metadata field branch_code is missing", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.Metadata = map[string]string{}
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("required metadata field branch_code is missing"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - required metadata field currency is missing", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.Metadata = map[string]string{
				client.GocardlessBranchCodeMetadataKey: "12345",
			}
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("required metadata field currency is missing"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - currency not supported", func(ctx SpecContext) {
			ba := sampleBankAccount

			ba.Metadata = map[string]string{
				client.GocardlessBranchCodeMetadataKey:  "12345",
				client.GocardlessCurrencyMetadataKey:    "XYZ",
				client.GocardlessAccountTypeMetadataKey: "savings",
			}
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)

			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("currency XYZ not supported"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - customer ID format invalid", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.Metadata = map[string]string{
				client.GocardlessBranchCodeMetadataKey:  "12345",
				client.GocardlessCurrencyMetadataKey:    "USD",
				client.GocardlessCustomerMetadataKey:    "INVALID123",
				client.GocardlessAccountTypeMetadataKey: "savings",
			}
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("customer ID must start with 'CU'"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - creditor ID format invalid", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.Metadata = map[string]string{
				client.GocardlessBranchCodeMetadataKey:  "12345",
				client.GocardlessCurrencyMetadataKey:    "USD",
				client.GocardlessCreditorMetadataKey:    "INVALID123",
				client.GocardlessAccountTypeMetadataKey: "savings",
			}
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("creditor ID must start with 'CR'"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - missing both customer and creditor", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.Metadata = map[string]string{
				client.GocardlessBranchCodeMetadataKey:  "12345",
				client.GocardlessCurrencyMetadataKey:    "USD",
				client.GocardlessAccountTypeMetadataKey: "savings",
			}
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("you must provide customer or creditor metadata field"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - you must provide either customer or creditor metadata field but not both", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.Metadata = map[string]string{
				client.GocardlessBranchCodeMetadataKey:  "12345",
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
			Expect(err).To(MatchError("you must provide either customer or creditor metadata field but not both"))
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
					m.EXPECT().CreateCreditorBankAccount(gomock.Any(), "CR123", ba).Return(expectedAccount, nil)
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
					m.EXPECT().CreateCustomerBankAccount(gomock.Any(), "CU123", ba).Return(expectedAccount, nil)
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
					client.GocardlessBranchCodeMetadataKey:  "12345",
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
					client.GocardlessBranchCodeMetadataKey:  "12345",
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
})
