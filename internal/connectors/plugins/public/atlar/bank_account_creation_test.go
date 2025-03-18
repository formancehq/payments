package atlar

import (
	"errors"
	"time"

	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/atlar/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/get-momo/atlar-v1-go-client/client/counterparties"
	atlar_models "github.com/get-momo/atlar-v1-go-client/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Atlar Plugin Bank Account Creation", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("create bank account from bank accout models", func() {
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
				IBAN:          pointer.For("FR9412739000405993414979X56"),
				SwiftBicCode:  pointer.For("ERAHJP6BT1H"),
				Country:       pointer.For("FR"),
				Metadata: map[string]string{
					"com.atlar.spec/owner/name": "test",
					"com.atlar.spec/owner/type": "INDIVIDUAL",
				},
			}
		})

		It("should return an error - missing owner name in bank account metadata", func(ctx SpecContext) {
			ba := sampleBankAccount
			delete(ba.Metadata, "com.atlar.spec/owner/name")
			req := models.CreateBankAccountRequest{
				BankAccount: &ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("required metadata field com.atlar.spec/owner/name is missing: invalid request"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - missing owner type in bank account metadata", func(ctx SpecContext) {
			ba := sampleBankAccount
			delete(ba.Metadata, "com.atlar.spec/owner/type")
			req := models.CreateBankAccountRequest{
				BankAccount: &ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("required metadata field com.atlar.spec/owner/type is missing: invalid request"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - wrong owner type in bank account metadata", func(ctx SpecContext) {
			ba := sampleBankAccount
			ba.Metadata["com.atlar.spec/owner/type"] = "WRONG"
			req := models.CreateBankAccountRequest{
				BankAccount: &ba,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("metadata field com.atlar.spec/owner/type needs to be one of [ INDIVIDUAL COMPANY ]: invalid request"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - create account error", func(ctx SpecContext) {
			ba := sampleBankAccount
			req := models.CreateBankAccountRequest{
				BankAccount: &ba,
			}

			m.EXPECT().PostV1CounterParties(ctx, gomock.Any()).Return(nil, errors.New("test-error"))

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test-error"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should work", func(ctx SpecContext) {
			ba := sampleBankAccount
			req := models.CreateBankAccountRequest{
				BankAccount: &ba,
			}

			m.EXPECT().PostV1CounterParties(ctx, atlar_models.CreateCounterpartyRequest{
				ContactDetails: &atlar_models.ContactDetails{
					Address: &atlar_models.Address{},
				},
				ExternalAccounts: []*atlar_models.CreateEmbeddedExternalAccountRequest{
					{
						Bank: &atlar_models.UpdatableBank{
							Bic: "ERAHJP6BT1H",
						},
						Identifiers: []*atlar_models.AccountIdentifier{
							{
								HolderName: pointer.For("test"),
								Market:     pointer.For("FR"),
								Number:     pointer.For("FR9412739000405993414979X56"),
								Type:       pointer.For("IBAN"),
							},
						},
					},
				},
				Name:      pointer.For("test"),
				PartyType: "INDIVIDUAL",
			}).Return(
				&counterparties.PostV1CounterpartiesCreated{
					Payload: &atlar_models.Counterparty{
						ContactDetails: &atlar_models.ContactDetails{
							Address: &atlar_models.Address{
								City:            "",
								Country:         "",
								PostalCode:      "",
								RawAddressLines: []string{},
								StreetName:      "",
								StreetNumber:    "",
							},
							Email:      "",
							NationalID: "",
							Phone:      "",
						},
						Created: new(string),
						ExternalAccounts: []*atlar_models.ExternalAccount{
							{
								Bank: &atlar_models.BankSlim{
									Bic:  "",
									ID:   "",
									Name: "",
								},
								CounterpartyID:   "",
								Created:          now.Format(time.RFC3339Nano),
								ExternalID:       "",
								ExternalMetadata: map[string]string{},
								ID:               "test",
								Identifiers: []*atlar_models.AccountIdentifier{
									{
										HolderName: pointer.For("test"),
										Market:     pointer.For("test"),
										Number:     pointer.For("test"),
										Type:       pointer.For("test"),
									},
								},
								OrganizationID: "",
								Updated:        "",
							},
						},
						ExternalID:       "",
						ExternalMetadata: map[string]string{},
						ID:               pointer.For("test"),
						Identifiers: []*atlar_models.AccountIdentifier{
							{
								HolderName: pointer.For("test"),
								Market:     pointer.For("test"),
								Number:     pointer.For("test"),
								Type:       pointer.For("test"),
							},
						},
						Name:           "",
						OrganizationID: new(string),
						PartyType:      "",
						Updated:        "",
						Version:        0,
					},
				},
				nil,
			)

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.RelatedAccount.Reference).To(Equal("test"))
		})
	})

	Context("create bank account from counter party models", func() {
		var (
			m                  *client.MockClient
			sampleBankAccount  models.BankAccount
			sampleCounterParty models.PSPCounterParty
			now                time.Time
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
					"com.atlar.spec/owner/name": "test",
					"com.atlar.spec/owner/type": "INDIVIDUAL",
				},
			}

			sampleCounterParty = models.PSPCounterParty{
				ID:        uuid.New(),
				Name:      "test",
				CreatedAt: now.UTC(),
				ContactDetails: &models.ContactDetails{
					Email: pointer.For("test"),
					Phone: pointer.For("0612345678"),
				},
				Address: &models.Address{
					StreetName:   pointer.For("test"),
					StreetNumber: pointer.For("1"),
					City:         pointer.For("test"),
					PostalCode:   pointer.For("12345"),
					Country:      pointer.For("FR"),
				},
				BankAccount: &sampleBankAccount,
				Metadata: map[string]string{
					"com.atlar.spec/owner/type": "INDIVIDUAL",
				},
			}
		})

		It("should return an error - empty owner name", func(ctx SpecContext) {
			cp := sampleCounterParty
			cp.Name = ""
			req := models.CreateBankAccountRequest{
				CounterParty: &cp,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("counter party name is required: invalid request"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - missing owner type in counter party metadata", func(ctx SpecContext) {
			cp := sampleCounterParty
			delete(cp.Metadata, "com.atlar.spec/owner/type")
			req := models.CreateBankAccountRequest{
				CounterParty: &cp,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("required metadata field com.atlar.spec/owner/type is missing: invalid request"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - wrong owner type in counter party metadata", func(ctx SpecContext) {
			cp := sampleCounterParty
			cp.Metadata["com.atlar.spec/owner/type"] = "WRONG"
			req := models.CreateBankAccountRequest{
				CounterParty: &cp,
			}

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("metadata field com.atlar.spec/owner/type needs to be one of [ INDIVIDUAL COMPANY ]: invalid request"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - create account error", func(ctx SpecContext) {
			cp := sampleCounterParty
			req := models.CreateBankAccountRequest{
				CounterParty: &cp,
			}

			m.EXPECT().PostV1CounterParties(ctx, gomock.Any()).Return(nil, errors.New("test-error"))

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test-error"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should work", func(ctx SpecContext) {
			cp := sampleCounterParty
			req := models.CreateBankAccountRequest{
				CounterParty: &cp,
			}

			m.EXPECT().PostV1CounterParties(ctx, atlar_models.CreateCounterpartyRequest{
				ContactDetails: &atlar_models.ContactDetails{
					Email: "test",
					Phone: "0612345678",
					Address: &atlar_models.Address{
						City:         "test",
						Country:      "FR",
						PostalCode:   "12345",
						StreetName:   "test",
						StreetNumber: "1",
					},
				},
				ExternalAccounts: []*atlar_models.CreateEmbeddedExternalAccountRequest{
					{
						Bank: &atlar_models.UpdatableBank{
							Bic: "ERAHJP6BT1H",
						},
						Identifiers: []*atlar_models.AccountIdentifier{
							{
								HolderName: pointer.For("test"),
								Market:     pointer.For("FR"),
								Number:     pointer.For("FR9412739000405993414979X56"),
								Type:       pointer.For("IBAN"),
							},
						},
					},
				},
				Name:      pointer.For("test"),
				PartyType: "INDIVIDUAL",
			}).Return(
				&counterparties.PostV1CounterpartiesCreated{
					Payload: &atlar_models.Counterparty{
						ContactDetails: &atlar_models.ContactDetails{
							Address: &atlar_models.Address{
								City:            "",
								Country:         "",
								PostalCode:      "",
								RawAddressLines: []string{},
								StreetName:      "",
								StreetNumber:    "",
							},
							Email:      "",
							NationalID: "",
							Phone:      "",
						},
						Created: new(string),
						ExternalAccounts: []*atlar_models.ExternalAccount{
							{
								Bank: &atlar_models.BankSlim{
									Bic:  "",
									ID:   "",
									Name: "",
								},
								CounterpartyID:   "",
								Created:          now.Format(time.RFC3339Nano),
								ExternalID:       "",
								ExternalMetadata: map[string]string{},
								ID:               "test",
								Identifiers: []*atlar_models.AccountIdentifier{
									{
										HolderName: pointer.For("test"),
										Market:     pointer.For("test"),
										Number:     pointer.For("test"),
										Type:       pointer.For("test"),
									},
								},
								OrganizationID: "",
								Updated:        "",
							},
						},
						ExternalID:       "",
						ExternalMetadata: map[string]string{},
						ID:               pointer.For("test"),
						Identifiers: []*atlar_models.AccountIdentifier{
							{
								HolderName: pointer.For("test"),
								Market:     pointer.For("test"),
								Number:     pointer.For("test"),
								Type:       pointer.For("test"),
							},
						},
						Name:           "",
						OrganizationID: new(string),
						PartyType:      "",
						Updated:        "",
						Version:        0,
					},
				},
				nil,
			)

			res, err := plg.CreateBankAccount(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.RelatedAccount.Reference).To(Equal("test"))
		})
	})
})
