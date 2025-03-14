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
			Expect(err).To(MatchError("required metadata field com.atlar.spec/owner/name is missing"))
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
			Expect(err).To(MatchError("required metadata field com.atlar.spec/owner/type is missing"))
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
			Expect(err).To(MatchError("metadata field com.atlar.spec/owner/type needs to be one of [ INDIVIDUAL COMPANY ]"))
			Expect(res).To(Equal(models.CreateBankAccountResponse{}))
		})

		It("should return an error - create account error", func(ctx SpecContext) {
			ba := sampleBankAccount
			req := models.CreateBankAccountRequest{
				BankAccount: &ba,
			}

			m.EXPECT().PostV1CounterParties(ctx, &ba).Return(nil, errors.New("test-error"))

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

			m.EXPECT().PostV1CounterParties(ctx, &ba).Return(
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
