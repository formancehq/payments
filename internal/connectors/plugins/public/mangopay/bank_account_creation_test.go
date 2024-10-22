package mangopay

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/mangopay/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Mangopay Plugin Bank Account Creation", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("create bank account", func() {
		var (
			m                   *client.MockClient
			sampleBankAccount   models.BankAccount
			sampleClientAddress client.OwnerAddress
			now                 time.Time
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
				IBAN:          pointer.For("FR9412739000405993414979X56"),
				SwiftBicCode:  pointer.For("ERAHJP6BT1H"),
				Country:       pointer.For("FR"),
				Metadata: map[string]string{
					models.BankAccountOwnerAddressLine1MetadataKey: "address1",
					models.BankAccountOwnerAddressLine2MetadataKey: "address2",
					models.BankAccountOwnerCityMetadataKey:         "city",
					models.BankAccountOwnerRegionMetadataKey:       "region",
					models.BankAccountOwnerPostalCodeMetadataKey:   "postal_code",
					client.MangopayUserIDMetadataKey:               "u1",
					client.MangopayTagMetadataKey:                  "foo=bar",
					client.MangopayABAMetadataKey:                  "aba",
					client.MangopayDepositAccountTypeMetadataKey:   "deposit_test",
					client.MangopayInstitutionNumberMetadataKey:    "institution_number_test",
					client.MangopayBranchCodeMetadataKey:           "branch_code",
					client.MangopayBankNameMetadataKey:             "bank_name",
					client.MangopaySortCodeMetadataKey:             "sort_code",
				},
			}

			sampleClientAddress = client.OwnerAddress{
				AddressLine1: "address1",
				AddressLine2: "address2",
				City:         "city",
				Region:       "region",
				PostalCode:   "postal_code",
				Country:      "FR",
			}
		})

		It("should return an error - missing userID in bank account metadata", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.Metadata = map[string]string{}
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("missing userID in bank account metadata"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - create IBAN bank account error", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.AccountNumber = nil
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			m.EXPECT().CreateIBANBankAccount(ctx, "u1", &client.CreateIBANBankAccountRequest{
				OwnerName:    ba.Name,
				OwnerAddress: &sampleClientAddress,
				IBAN:         *ba.IBAN,
				BIC:          *ba.SwiftBicCode,
				Tag:          "foo=bar",
			}).Return(nil, errors.New("test error"))

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to create account: test error"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should be ok - create IBAN bank account", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.AccountNumber = nil
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			expectedBA := &client.BankAccount{
				ID:           "id",
				OwnerName:    ba.Name,
				CreationDate: now.UTC().Unix(),
			}
			m.EXPECT().CreateIBANBankAccount(ctx, "u1", &client.CreateIBANBankAccountRequest{
				OwnerName:    ba.Name,
				OwnerAddress: &sampleClientAddress,
				IBAN:         *ba.IBAN,
				BIC:          *ba.SwiftBicCode,
				Tag:          "foo=bar",
			}).Return(expectedBA, nil)

			raw, _ := json.Marshal(expectedBA)

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(models.CreateBankAccountResponse{
				RelatedAccount: models.PSPAccount{
					Reference: "id",
					CreatedAt: time.Unix(expectedBA.CreationDate, 0),
					Name:      &ba.Name,
					Metadata: map[string]string{
						"user_id": "u1",
					},
					Raw: raw,
				},
			}))
		})

		It("should return an error - create us bank account with missing account number", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.IBAN = nil
			ba.AccountNumber = nil
			ba.Country = pointer.For("US")
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("missing account number in request"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - createUSBankAccount error", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.IBAN = nil
			ba.Country = pointer.For("US")
			sca := sampleClientAddress
			sca.Country = "US"
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			m.EXPECT().CreateUSBankAccount(ctx, "u1", &client.CreateUSBankAccountRequest{
				OwnerName:          ba.Name,
				OwnerAddress:       &sca,
				AccountNumber:      *ba.AccountNumber,
				ABA:                "aba",
				DepositAccountType: "deposit_test",
				Tag:                "foo=bar",
			}).Return(nil, errors.New("test error"))

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to create account: test error"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should be ok - create US bank account", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.IBAN = nil
			ba.Country = pointer.For("US")
			sca := sampleClientAddress
			sca.Country = "US"
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			expectedBA := &client.BankAccount{
				ID:           "id",
				OwnerName:    ba.Name,
				CreationDate: now.UTC().Unix(),
			}
			m.EXPECT().CreateUSBankAccount(ctx, "u1", &client.CreateUSBankAccountRequest{
				OwnerName:          ba.Name,
				OwnerAddress:       &sca,
				AccountNumber:      *ba.AccountNumber,
				ABA:                "aba",
				DepositAccountType: "deposit_test",
				Tag:                "foo=bar",
			}).Return(expectedBA, nil)

			raw, _ := json.Marshal(expectedBA)

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(models.CreateBankAccountResponse{
				RelatedAccount: models.PSPAccount{
					Reference: "id",
					CreatedAt: time.Unix(expectedBA.CreationDate, 0),
					Name:      &ba.Name,
					Metadata: map[string]string{
						"user_id": "u1",
					},
					Raw: raw,
				},
			}))
		})

		It("should return an error - create CA bank account with missing account number", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.IBAN = nil
			ba.AccountNumber = nil
			ba.Country = pointer.For("CA")
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("missing account number in request"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - createCABankAccount error", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.IBAN = nil
			ba.Country = pointer.For("CA")
			sca := sampleClientAddress
			sca.Country = "CA"
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			m.EXPECT().CreateCABankAccount(ctx, "u1", &client.CreateCABankAccountRequest{
				OwnerName:         ba.Name,
				OwnerAddress:      &sca,
				AccountNumber:     *ba.AccountNumber,
				InstitutionNumber: "institution_number_test",
				BranchCode:        "branch_code",
				BankName:          "bank_name",
				Tag:               "foo=bar",
			}).Return(nil, errors.New("test error"))

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to create account: test error"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should be ok - create CA bank account", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.IBAN = nil
			ba.Country = pointer.For("CA")
			sca := sampleClientAddress
			sca.Country = "CA"
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			expectedBA := &client.BankAccount{
				ID:           "id",
				OwnerName:    ba.Name,
				CreationDate: now.UTC().Unix(),
			}
			m.EXPECT().CreateCABankAccount(ctx, "u1", &client.CreateCABankAccountRequest{
				OwnerName:         ba.Name,
				OwnerAddress:      &sca,
				AccountNumber:     *ba.AccountNumber,
				InstitutionNumber: "institution_number_test",
				BranchCode:        "branch_code",
				BankName:          "bank_name",
				Tag:               "foo=bar",
			}).Return(expectedBA, nil)

			raw, _ := json.Marshal(expectedBA)

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(models.CreateBankAccountResponse{
				RelatedAccount: models.PSPAccount{
					Reference: "id",
					CreatedAt: time.Unix(expectedBA.CreationDate, 0),
					Name:      &ba.Name,
					Metadata: map[string]string{
						"user_id": "u1",
					},
					Raw: raw,
				},
			}))
		})

		It("should return an error - create GB bank account with missing account number", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.IBAN = nil
			ba.AccountNumber = nil
			ba.Country = pointer.For("GB")
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("missing account number in request"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - createGBBankAccount error", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.IBAN = nil
			ba.Country = pointer.For("GB")
			sca := sampleClientAddress
			sca.Country = "GB"
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			m.EXPECT().CreateGBBankAccount(ctx, "u1", &client.CreateGBBankAccountRequest{
				OwnerName:     ba.Name,
				OwnerAddress:  &sca,
				AccountNumber: *ba.AccountNumber,
				SortCode:      "sort_code",
				Tag:           "foo=bar",
			}).Return(nil, errors.New("test error"))

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to create account: test error"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should be ok - create GB bank account", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.IBAN = nil
			ba.Country = pointer.For("GB")
			sca := sampleClientAddress
			sca.Country = "GB"
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			expectedBA := &client.BankAccount{
				ID:           "id",
				OwnerName:    ba.Name,
				CreationDate: now.UTC().Unix(),
			}
			m.EXPECT().CreateGBBankAccount(ctx, "u1", &client.CreateGBBankAccountRequest{
				OwnerName:     ba.Name,
				OwnerAddress:  &sca,
				AccountNumber: *ba.AccountNumber,
				SortCode:      "sort_code",
				Tag:           "foo=bar",
			}).Return(expectedBA, nil)

			raw, _ := json.Marshal(expectedBA)

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(models.CreateBankAccountResponse{
				RelatedAccount: models.PSPAccount{
					Reference: "id",
					CreatedAt: time.Unix(expectedBA.CreationDate, 0),
					Name:      &ba.Name,
					Metadata: map[string]string{
						"user_id": "u1",
					},
					Raw: raw,
				},
			}))
		})

		It("should return an error - create other bank account with missing account number", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.IBAN = nil
			ba.AccountNumber = nil
			ba.Country = pointer.For("TEST_COUNTRY")
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("missing account number in request"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - createOtherBankAccount error", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.IBAN = nil
			ba.Country = pointer.For("TEST_COUNTRY")
			sca := sampleClientAddress
			sca.Country = "TEST_COUNTRY"
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			m.EXPECT().CreateOtherBankAccount(ctx, "u1", &client.CreateOtherBankAccountRequest{
				OwnerName:     ba.Name,
				OwnerAddress:  &sca,
				AccountNumber: *ba.AccountNumber,
				BIC:           *ba.SwiftBicCode,
				Country:       "TEST_COUNTRY",
				Tag:           "foo=bar",
			}).Return(nil, errors.New("test error"))

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to create account: test error"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should be ok - create other bank account", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.IBAN = nil
			ba.Country = pointer.For("TEST_COUNTRY")
			sca := sampleClientAddress
			sca.Country = "TEST_COUNTRY"
			req := models.CreateBankAccountRequest{
				BankAccount: ba,
			}

			expectedBA := &client.BankAccount{
				ID:           "id",
				OwnerName:    ba.Name,
				CreationDate: now.UTC().Unix(),
			}
			m.EXPECT().CreateOtherBankAccount(ctx, "u1", &client.CreateOtherBankAccountRequest{
				OwnerName:     ba.Name,
				OwnerAddress:  &sca,
				AccountNumber: *ba.AccountNumber,
				BIC:           *ba.SwiftBicCode,
				Country:       "TEST_COUNTRY",
				Tag:           "foo=bar",
			}).Return(expectedBA, nil)

			raw, _ := json.Marshal(expectedBA)

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(models.CreateBankAccountResponse{
				RelatedAccount: models.PSPAccount{
					Reference: "id",
					CreatedAt: time.Unix(expectedBA.CreationDate, 0),
					Name:      &ba.Name,
					Metadata: map[string]string{
						"user_id": "u1",
					},
					Raw: raw,
				},
			}))
		})
	})
})
