package mangopay

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/pkg/connectors/mangopay/client"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Mangopay Plugin Bank Account Creation", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  connector.Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("create bank account", func() {
		var (
			sampleBankAccount   connector.BankAccount
			sampleClientAddress client.OwnerAddress
			now                 time.Time
		)

		BeforeEach(func() {
			now = time.Now().UTC()

			sampleBankAccount = connector.BankAccount{
				ID:            uuid.New(),
				CreatedAt:     now.UTC(),
				Name:          "test",
				AccountNumber: pointer.For("12345678"),
				IBAN:          pointer.For("FR9412739000405993414979X56"),
				SwiftBicCode:  pointer.For("ERAHJP6BT1H"),
				Country:       pointer.For("FR"),
				Metadata: map[string]string{
					connector.BankAccountOwnerAddressLine1MetadataKey: "address1",
					connector.BankAccountOwnerAddressLine2MetadataKey: "address2",
					connector.BankAccountOwnerCityMetadataKey:         "city",
					connector.BankAccountOwnerRegionMetadataKey:       "region",
					connector.BankAccountOwnerPostalCodeMetadataKey:   "postal_code",
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
			req := connector.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(connector.ErrMissingConnectorMetadata))
			Expect(res).To(Equal(connector.CreateBankAccountResponse{}))
		})

		It("should return an error - create IBAN bank account error", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.AccountNumber = nil
			req := connector.CreateBankAccountRequest{
				BankAccount: ba,
			}

			m.EXPECT().CreateIBANBankAccount(gomock.Any(), "u1", &client.CreateIBANBankAccountRequest{
				OwnerName:    ba.Name,
				OwnerAddress: &sampleClientAddress,
				IBAN:         *ba.IBAN,
				BIC:          *ba.SwiftBicCode,
				Tag:          "foo=bar",
			}).Return(nil, errors.New("test error"))

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to create IBAN bank account: test error: failed to create account"))
			Expect(res).To(Equal(connector.CreateBankAccountResponse{}))
		})

		It("should be ok - create IBAN bank account", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.AccountNumber = nil
			req := connector.CreateBankAccountRequest{
				BankAccount: ba,
			}

			expectedBA := &client.BankAccount{
				ID:           "id",
				OwnerName:    ba.Name,
				CreationDate: now.UTC().Unix(),
			}
			m.EXPECT().CreateIBANBankAccount(gomock.Any(), "u1", &client.CreateIBANBankAccountRequest{
				OwnerName:    ba.Name,
				OwnerAddress: &sampleClientAddress,
				IBAN:         *ba.IBAN,
				BIC:          *ba.SwiftBicCode,
				Tag:          "foo=bar",
			}).Return(expectedBA, nil)

			raw, _ := json.Marshal(expectedBA)

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(connector.CreateBankAccountResponse{
				RelatedAccount: connector.PSPAccount{
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
			req := connector.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("missing account number in request"))
			Expect(res).To(Equal(connector.CreateBankAccountResponse{}))
		})

		It("should return an error - createUSBankAccount error", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.IBAN = nil
			ba.Country = pointer.For("US")
			sca := sampleClientAddress
			sca.Country = "US"
			req := connector.CreateBankAccountRequest{
				BankAccount: ba,
			}

			m.EXPECT().CreateUSBankAccount(gomock.Any(), "u1", &client.CreateUSBankAccountRequest{
				OwnerName:          ba.Name,
				OwnerAddress:       &sca,
				AccountNumber:      *ba.AccountNumber,
				ABA:                "aba",
				DepositAccountType: "deposit_test",
				Tag:                "foo=bar",
			}).Return(nil, errors.New("test error"))

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to create US bank account: test error: failed to create account"))
			Expect(res).To(Equal(connector.CreateBankAccountResponse{}))
		})

		It("should be ok - create US bank account", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.IBAN = nil
			ba.Country = pointer.For("US")
			sca := sampleClientAddress
			sca.Country = "US"
			req := connector.CreateBankAccountRequest{
				BankAccount: ba,
			}

			expectedBA := &client.BankAccount{
				ID:           "id",
				OwnerName:    ba.Name,
				CreationDate: now.UTC().Unix(),
			}
			m.EXPECT().CreateUSBankAccount(gomock.Any(), "u1", &client.CreateUSBankAccountRequest{
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
			Expect(res).To(Equal(connector.CreateBankAccountResponse{
				RelatedAccount: connector.PSPAccount{
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
			req := connector.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("missing account number in request"))
			Expect(res).To(Equal(connector.CreateBankAccountResponse{}))
		})

		It("should return an error - createCABankAccount error", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.IBAN = nil
			ba.Country = pointer.For("CA")
			sca := sampleClientAddress
			sca.Country = "CA"
			req := connector.CreateBankAccountRequest{
				BankAccount: ba,
			}

			m.EXPECT().CreateCABankAccount(gomock.Any(), "u1", &client.CreateCABankAccountRequest{
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
			Expect(err).To(MatchError("failed to create CA bank account: test error: failed to create account"))
			Expect(res).To(Equal(connector.CreateBankAccountResponse{}))
		})

		It("should be ok - create CA bank account", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.IBAN = nil
			ba.Country = pointer.For("CA")
			sca := sampleClientAddress
			sca.Country = "CA"
			req := connector.CreateBankAccountRequest{
				BankAccount: ba,
			}

			expectedBA := &client.BankAccount{
				ID:           "id",
				OwnerName:    ba.Name,
				CreationDate: now.UTC().Unix(),
			}
			m.EXPECT().CreateCABankAccount(gomock.Any(), "u1", &client.CreateCABankAccountRequest{
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
			Expect(res).To(Equal(connector.CreateBankAccountResponse{
				RelatedAccount: connector.PSPAccount{
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
			req := connector.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("missing account number in request"))
			Expect(res).To(Equal(connector.CreateBankAccountResponse{}))
		})

		It("should return an error - createGBBankAccount error", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.IBAN = nil
			ba.Country = pointer.For("GB")
			sca := sampleClientAddress
			sca.Country = "GB"
			req := connector.CreateBankAccountRequest{
				BankAccount: ba,
			}

			m.EXPECT().CreateGBBankAccount(gomock.Any(), "u1", &client.CreateGBBankAccountRequest{
				OwnerName:     ba.Name,
				OwnerAddress:  &sca,
				AccountNumber: *ba.AccountNumber,
				SortCode:      "sort_code",
				Tag:           "foo=bar",
			}).Return(nil, errors.New("test error"))

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to create GB bank account: test error: failed to create account"))
			Expect(res).To(Equal(connector.CreateBankAccountResponse{}))
		})

		It("should be ok - create GB bank account", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.IBAN = nil
			ba.Country = pointer.For("GB")
			sca := sampleClientAddress
			sca.Country = "GB"
			req := connector.CreateBankAccountRequest{
				BankAccount: ba,
			}

			expectedBA := &client.BankAccount{
				ID:           "id",
				OwnerName:    ba.Name,
				CreationDate: now.UTC().Unix(),
			}
			m.EXPECT().CreateGBBankAccount(gomock.Any(), "u1", &client.CreateGBBankAccountRequest{
				OwnerName:     ba.Name,
				OwnerAddress:  &sca,
				AccountNumber: *ba.AccountNumber,
				SortCode:      "sort_code",
				Tag:           "foo=bar",
			}).Return(expectedBA, nil)

			raw, _ := json.Marshal(expectedBA)

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(connector.CreateBankAccountResponse{
				RelatedAccount: connector.PSPAccount{
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
			req := connector.CreateBankAccountRequest{
				BankAccount: ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("missing account number in request"))
			Expect(res).To(Equal(connector.CreateBankAccountResponse{}))
		})

		It("should return an error - createOtherBankAccount error", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.IBAN = nil
			ba.Country = pointer.For("TEST_COUNTRY")
			sca := sampleClientAddress
			sca.Country = "TEST_COUNTRY"
			req := connector.CreateBankAccountRequest{
				BankAccount: ba,
			}

			m.EXPECT().CreateOtherBankAccount(gomock.Any(), "u1", &client.CreateOtherBankAccountRequest{
				OwnerName:     ba.Name,
				OwnerAddress:  &sca,
				AccountNumber: *ba.AccountNumber,
				BIC:           *ba.SwiftBicCode,
				Country:       "TEST_COUNTRY",
				Tag:           "foo=bar",
			}).Return(nil, errors.New("test error"))

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to create other bank account: test error: failed to create account"))
			Expect(res).To(Equal(connector.CreateBankAccountResponse{}))
		})

		It("should be ok - create other bank account", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.IBAN = nil
			ba.Country = pointer.For("TEST_COUNTRY")
			sca := sampleClientAddress
			sca.Country = "TEST_COUNTRY"
			req := connector.CreateBankAccountRequest{
				BankAccount: ba,
			}

			expectedBA := &client.BankAccount{
				ID:           "id",
				OwnerName:    ba.Name,
				CreationDate: now.UTC().Unix(),
			}
			m.EXPECT().CreateOtherBankAccount(gomock.Any(), "u1", &client.CreateOtherBankAccountRequest{
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
			Expect(res).To(Equal(connector.CreateBankAccountResponse{
				RelatedAccount: connector.PSPAccount{
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
