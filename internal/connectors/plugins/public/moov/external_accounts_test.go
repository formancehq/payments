package moov

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/moov/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/moovfinancial/moov-go/pkg/moov"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

// Error returned by the custom marshaller
var errCustomMarshal = errors.New("custom marshal error")

// Type for testing Marshal errors
type UnmarshalableBankAccount struct {
	moov.BankAccount
}

// MarshalJSON always returns an error
func (u UnmarshalableBankAccount) MarshalJSON() ([]byte, error) {
	return nil, errCustomMarshal
}

var _ = Describe("Error handling when fillExternalAccounts fails", func() {
	It("should handle errors from fillExternalAccounts", func(ctx SpecContext) {

		ctrl := gomock.NewController(GinkgoT())
		mockClient := client.NewMockClient(ctrl)

		sampleAccount := moov.Account{AccountID: "account123"}
		accountPayload, _ := json.Marshal(sampleAccount)
		req := models.FetchNextExternalAccountsRequest{
			FromPayload: accountPayload,
		}

		bankAccounts := []moov.BankAccount{
			{BankAccountID: "bank1"},
		}
		mockClient.EXPECT().GetExternalAccounts(gomock.Any(), sampleAccount.AccountID).Return(
			bankAccounts,
			nil,
		)

		fetchNextWithError := func(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
			var from moov.Account
			if req.FromPayload != nil {
				if err := json.Unmarshal(req.FromPayload, &from); err != nil {
					return models.FetchNextExternalAccountsResponse{}, err
				}
			}

			bankAccounts, err := mockClient.GetExternalAccounts(ctx, from.AccountID)
			if err != nil {
				return models.FetchNextExternalAccountsResponse{}, err
			}

			Expect(bankAccounts).To(HaveLen(1))
			Expect(bankAccounts[0].BankAccountID).To(Equal("bank1"))

			return models.FetchNextExternalAccountsResponse{}, errors.New("fill external accounts error")
		}

		resp, err := fetchNextWithError(ctx, req)

		Expect(err).To(MatchError("fill external accounts error"))
		Expect(resp.ExternalAccounts).To(HaveLen(0))
	})
})

var _ = Describe("Moov External Accounts", func() {
	var (
		plg                *Plugin
		sampleMoovAccount  moov.Account
		sampleBankAccounts []moov.BankAccount
	)

	BeforeEach(func() {
		plg = &Plugin{}
		sampleMoovAccount = moov.Account{
			AccountID:   "account123",
			DisplayName: "Test Account",
			CreatedOn:   time.Now().UTC(),
		}

		// Setup sample bank accounts - simplified to avoid ExceptionDetails
		sampleBankAccounts = []moov.BankAccount{
			{
				BankAccountID:         "bank1",
				HolderName:            "John Doe",
				HolderType:            "individual",
				BankName:              "Test Bank",
				BankAccountType:       "checking",
				RoutingNumber:         "123456789",
				LastFourAccountNumber: "1234",
				Status:                "active",
				StatusReason:          "activated",
				Fingerprint:           "fingerprint123",
				UpdatedOn:             time.Now().UTC(),
			},
			{
				BankAccountID:         "bank2",
				HolderName:            "Jane Smith",
				HolderType:            "business",
				BankName:              "Other Bank",
				BankAccountType:       "savings",
				RoutingNumber:         "987654321",
				LastFourAccountNumber: "4321",
				Status:                "active",
				StatusReason:          "activated",
				Fingerprint:           "fingerprint456",
				UpdatedOn:             time.Now().UTC(),
			},
		}
	})

	Context("fetching next external accounts", func() {
		var (
			m *client.MockClient
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
		})

		It("should return ErrNotYetInstalled when client is nil", func(ctx SpecContext) {
			// Setting client to nil to trigger the error
			plg.client = nil

			req := models.FetchNextExternalAccountsRequest{
				FromPayload: nil,
			}

			resp, err := plg.FetchNextExternalAccounts(ctx, req)

			Expect(err).To(Equal(plugins.ErrNotYetInstalled))
			Expect(resp.ExternalAccounts).To(HaveLen(0))
		})

		It("should return an error - get external accounts error", func(ctx SpecContext) {
			accountPayload, _ := json.Marshal(sampleMoovAccount)
			req := models.FetchNextExternalAccountsRequest{
				FromPayload: accountPayload,
			}

			m.EXPECT().GetExternalAccounts(gomock.Any(), sampleMoovAccount.AccountID).Return(
				nil,
				errors.New("test error"),
			)

			resp, err := plg.fetchNextExternalAccounts(ctx, req)

			Expect(err).To(MatchError("test error"))
			Expect(resp.ExternalAccounts).To(HaveLen(0))
		})

		It("should return an error - invalid from payload", func(ctx SpecContext) {
			invalidPayload := []byte(`invalid json`)
			req := models.FetchNextExternalAccountsRequest{
				FromPayload: invalidPayload,
			}

			resp, err := plg.fetchNextExternalAccounts(ctx, req)

			Expect(err).NotTo(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(0))
		})

		It("should fetch external accounts successfully", func(ctx SpecContext) {
			accountPayload, _ := json.Marshal(sampleMoovAccount)
			req := models.FetchNextExternalAccountsRequest{
				FromPayload: accountPayload,
			}

			m.EXPECT().GetExternalAccounts(gomock.Any(), sampleMoovAccount.AccountID).Return(
				sampleBankAccounts,
				nil,
			)

			resp, err := plg.fetchNextExternalAccounts(ctx, req)

			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(2))
			Expect(resp.HasMore).To(BeFalse())

			// Verify first bank account
			Expect(resp.ExternalAccounts[0].Reference).To(Equal("bank1"))
			Expect(*resp.ExternalAccounts[0].Name).To(Equal("John Doe"))
			Expect(resp.ExternalAccounts[0].Metadata).To(HaveKeyWithValue(client.MoovBankNameMetadataKey, "Test Bank"))
			Expect(resp.ExternalAccounts[0].Metadata).To(HaveKeyWithValue(client.MoovHolderTypeMetadataKey, "individual"))
			Expect(resp.ExternalAccounts[0].Metadata).To(HaveKeyWithValue(client.MoovBankAccountTypeMetadataKey, "checking"))
			Expect(resp.ExternalAccounts[0].Metadata).To(HaveKeyWithValue(client.MoovRoutingNumberMetadataKey, "123456789"))
			Expect(resp.ExternalAccounts[0].Metadata).To(HaveKeyWithValue(client.MoovLastFourAccountNumberMetadataKey, "1234"))
			Expect(resp.ExternalAccounts[0].Metadata).To(HaveKeyWithValue(client.MoovStatusMetadataKey, "active"))
			Expect(resp.ExternalAccounts[0].Metadata).To(HaveKeyWithValue(client.MoovStatusReasonMetadataKey, "activated"))
			Expect(resp.ExternalAccounts[0].Metadata).To(HaveKeyWithValue(client.MoovFingerprintMetadataKey, "fingerprint123"))

			// Verify second bank account
			Expect(resp.ExternalAccounts[1].Reference).To(Equal("bank2"))
			Expect(*resp.ExternalAccounts[1].Name).To(Equal("Jane Smith"))
			Expect(resp.ExternalAccounts[1].Metadata).To(HaveKeyWithValue(client.MoovBankNameMetadataKey, "Other Bank"))
		})

		It("should fetch external accounts with nil FromPayload", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				FromPayload: nil,
			}

			m.EXPECT().GetExternalAccounts(gomock.Any(), "").Return(
				sampleBankAccounts,
				nil,
			)

			resp, err := plg.fetchNextExternalAccounts(ctx, req)

			Expect(err).To(BeNil())
			Expect(resp.ExternalAccounts).To(HaveLen(2))
			Expect(resp.HasMore).To(BeFalse())
		})

		Context("fetch external accounts with moov client", func() {
			var (
				mockedService *client.MockMoovClient
			)

			BeforeEach((func() {
				ctrl := gomock.NewController(GinkgoT())
				mockedService = client.NewMockMoovClient(ctrl)

				plg.client, _ = client.New("moov", "https://example.com", "access_token", "test", "test")
				plg.client.NewWithClient(mockedService)
			}))

			It("should fail when moov client returns an error", func(ctx SpecContext) {
				accountPayload, _ := json.Marshal(sampleMoovAccount)
				req := models.FetchNextExternalAccountsRequest{
					FromPayload: accountPayload,
				}

				mockedService.EXPECT().GetMoovBankAccounts(gomock.Any(), sampleMoovAccount.AccountID).Return(
					nil,
					errors.New("fetch bank accounts error"),
				)

				resp, err := plg.fetchNextExternalAccounts(ctx, req)

				Expect(err).To(MatchError("failed to get moov bank accounts: fetch bank accounts error"))
				Expect(resp.ExternalAccounts).To(HaveLen(0))
			})

			It("should successfully fetch external accounts using moov client", func(ctx SpecContext) {
				accountPayload, _ := json.Marshal(sampleMoovAccount)
				req := models.FetchNextExternalAccountsRequest{
					FromPayload: accountPayload,
				}

				mockedService.EXPECT().GetMoovBankAccounts(gomock.Any(), sampleMoovAccount.AccountID).Return(
					sampleBankAccounts,
					nil,
				)

				resp, err := plg.fetchNextExternalAccounts(ctx, req)

				Expect(err).To(BeNil())
				Expect(resp.ExternalAccounts).To(HaveLen(2))
				Expect(resp.HasMore).To(BeFalse())
			})
		})
	})

	Context("fillExternalAccounts", func() {
		BeforeEach(func() {
			plg = &Plugin{}
		})

		It("should convert bank accounts to PSP accounts correctly", func() {
			accounts, err := plg.fillExternalAccounts(sampleMoovAccount.AccountID, sampleBankAccounts)

			Expect(err).To(BeNil())
			Expect(accounts).To(HaveLen(2))

			// Verify first account
			Expect(accounts[0].Reference).To(Equal("bank1"))
			Expect(*accounts[0].Name).To(Equal("John Doe"))
			Expect(accounts[0].Metadata).To(HaveKeyWithValue(client.MoovBankNameMetadataKey, "Test Bank"))
			Expect(accounts[0].Metadata).To(HaveKeyWithValue(client.MoovHolderTypeMetadataKey, "individual"))
			Expect(accounts[0].Metadata).To(HaveKeyWithValue(client.MoovBankAccountTypeMetadataKey, "checking"))

			// Verify second account
			Expect(accounts[1].Reference).To(Equal("bank2"))
			Expect(*accounts[1].Name).To(Equal("Jane Smith"))
			Expect(accounts[1].Metadata).To(HaveKeyWithValue(client.MoovBankNameMetadataKey, "Other Bank"))
			Expect(accounts[1].Metadata).To(HaveKeyWithValue(client.MoovHolderTypeMetadataKey, "business"))
			Expect(accounts[1].Metadata).To(HaveKeyWithValue(client.MoovBankAccountTypeMetadataKey, "savings"))

			// Verify raw data
			var bankAccount moov.BankAccount
			err = json.Unmarshal(accounts[0].Raw, &bankAccount)
			Expect(err).To(BeNil())
			Expect(bankAccount.BankAccountID).To(Equal("bank1"))
		})

		It("should handle exception details correctly", func() {
			// Create a sample bank account with exception details
			achReturnCode := moov.AchReturnCode_R02
			rtpRejectionCode := moov.RTPRejectionCode_AC03

			bankAccountWithExceptions := moov.BankAccount{
				BankAccountID:         "bank3",
				HolderName:            "Alex Johnson",
				HolderType:            "individual",
				BankName:              "Exception Bank",
				BankAccountType:       "checking",
				RoutingNumber:         "111222333",
				LastFourAccountNumber: "5678",
				Status:                "error",
				StatusReason:          "failed_verification",
				Fingerprint:           "fingerprint789",
				UpdatedOn:             time.Now().UTC(),
				ExceptionDetails: &moov.ExceptionDetails{
					Description:      "Test exception description",
					AchReturnCode:    &achReturnCode,
					RTPRejectionCode: &rtpRejectionCode,
				},
			}

			bankAccountsWithExceptions := []moov.BankAccount{bankAccountWithExceptions}

			accounts, err := plg.fillExternalAccounts(sampleMoovAccount.AccountID, bankAccountsWithExceptions)

			Expect(err).To(BeNil())
			Expect(accounts).To(HaveLen(1))

			// Verify all basic metadata is set
			Expect(accounts[0].Reference).To(Equal("bank3"))
			Expect(*accounts[0].Name).To(Equal("Alex Johnson"))
			Expect(accounts[0].Metadata).To(HaveKeyWithValue(client.MoovBankNameMetadataKey, "Exception Bank"))

			// Verify exception details metadata is set correctly
			Expect(accounts[0].Metadata).To(HaveKeyWithValue(client.MoovExceptionDetailsDescriptionMetadataKey, "Test exception description"))
			Expect(accounts[0].Metadata).To(HaveKeyWithValue(client.MoovExceptionDetailsAchReturnCodeMetadataKey, string(achReturnCode)))
			Expect(accounts[0].Metadata).To(HaveKeyWithValue(client.MoovExceptionDetailsRTPRejectionCodeMetadataKey, string(rtpRejectionCode)))

			// Verify raw data contains the exception details
			var bankAccount moov.BankAccount
			err = json.Unmarshal(accounts[0].Raw, &bankAccount)
			Expect(err).To(BeNil())
			Expect(bankAccount.ExceptionDetails).NotTo(BeNil())
			Expect(bankAccount.ExceptionDetails.Description).To(Equal("Test exception description"))
			Expect(*bankAccount.ExceptionDetails.AchReturnCode).To(Equal(achReturnCode))
			Expect(*bankAccount.ExceptionDetails.RTPRejectionCode).To(Equal(rtpRejectionCode))
		})

	})
})
