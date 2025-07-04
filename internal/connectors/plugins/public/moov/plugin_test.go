package moov

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/moov/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/moovfinancial/moov-go/pkg/moov"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Moov Plugin Suite")
}

var _ = Describe("Moov Plugin", func() {
	var (
		plg    *Plugin
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		plg = &Plugin{
			name:   ProviderName,
			logger: logger,
		}
	})

	Context("plugin initialization", func() {
		It("reports validation errors in the config", func() {
			config := json.RawMessage(`{}`)
			_, err := New(ProviderName, logger, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("validation"))
		})

		It("should strip http:// prefix from endpoint", func() {
			config := json.RawMessage(`{
				"endpoint": "http://test.com",
				"publicKey": "test-public-key",
				"privateKey": "test-private-key",
				"accountID": "test-account-id"
			}`)
			plg, err := New(ProviderName, logger, config)
			Expect(err).NotTo(HaveOccurred())
			Expect(plg).NotTo(BeNil())
			Expect(plg.client).NotTo(BeNil())
		})

		It("should strip https:// prefix from endpoint", func() {
			config := json.RawMessage(`{
				"endpoint": "https://test.com",
				"publicKey": "test-public-key",
				"privateKey": "test-private-key",
				"accountID": "test-account-id"
			}`)
			plg, err := New(ProviderName, logger, config)
			Expect(err).NotTo(HaveOccurred())
			Expect(plg).NotTo(BeNil())
			Expect(plg.client).NotTo(BeNil())
		})

		It("should accept valid URL with port", func() {
			config := json.RawMessage(`{
				"endpoint": "https://api.test.com:8080",
				"publicKey": "test-public-key",
				"privateKey": "test-private-key",
				"accountID": "test-account-id"
			}`)
			plg, err := New(ProviderName, logger, config)
			Expect(err).NotTo(HaveOccurred())
			Expect(plg).NotTo(BeNil())
			Expect(plg.client).NotTo(BeNil())
		})

		It("should report errors when endpoint is missing", func() {
			config := json.RawMessage(`{
				"publicKey": "test-public-key",
				"privateKey": "test-private-key",
				"accountID": "test-account-id"
			}`)
			_, err := New(ProviderName, logger, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Endpoint"))
		})

		It("should report errors when publicKey is missing", func() {
			config := json.RawMessage(`{
				"endpoint": "http://test.com",
				"privateKey": "test-private-key",
				"accountID": "test-account-id"
			}`)
			_, err := New(ProviderName, logger, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("PublicKey"))
		})

		It("should report errors when privateKey is missing", func() {
			config := json.RawMessage(`{
				"endpoint": "http://test.com",
				"publicKey": "test-public-key",
				"accountID": "test-account-id"
			}`)
			_, err := New(ProviderName, logger, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("PrivateKey"))
		})

		It("should report errors when accountID is missing", func() {
			config := json.RawMessage(`{
				"endpoint": "http://test.com",
				"publicKey": "test-public-key",
				"privateKey": "test-private-key"
			}`)
			_, err := New(ProviderName, logger, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("AccountID"))
		})

		It("should create a valid plugin with complete config", func() {
			config := json.RawMessage(`{
				"endpoint": "http://test.com",
				"publicKey": "test-public-key",
				"privateKey": "test-private-key",
				"accountID": "test-account-id"
			}`)
			plg, err := New(ProviderName, logger, config)
			Expect(err).NotTo(HaveOccurred())
			Expect(plg).NotTo(BeNil())
			Expect(plg.client).NotTo(BeNil())
			Expect(plg.Name()).To(Equal(ProviderName))
		})
	})

	Context("unmarshalAndValidateConfig function", func() {

		It("should strip http:// prefix from endpoint", func() {
			config := json.RawMessage(`{
				"endpoint": "http://test.com",
				"publicKey": "test-public-key",
				"privateKey": "test-private-key",
				"accountID": "test-account-id"
			}`)
			result, err := unmarshalAndValidateConfig(config)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Endpoint).To(Equal("test.com"))
		})

		It("should strip https:// prefix from endpoint", func() {
			config := json.RawMessage(`{
				"endpoint": "https://test.com",
				"publicKey": "test-public-key",
				"privateKey": "test-private-key",
				"accountID": "test-account-id"
			}`)
			result, err := unmarshalAndValidateConfig(config)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Endpoint).To(Equal("test.com"))
		})

		It("should not accept URL without protocol", func() {
			config := json.RawMessage(`{
				"endpoint": "api.test.com",
				"publicKey": "test-public-key",
				"privateKey": "test-private-key",
				"accountID": "test-account-id"
			}`)
			_, err := unmarshalAndValidateConfig(config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("url"))
		})

		It("should accept valid URL with port and strip protocol", func() {
			config := json.RawMessage(`{
				"endpoint": "https://api.test.com:8080",
				"publicKey": "test-public-key",
				"privateKey": "test-private-key",
				"accountID": "test-account-id"
			}`)
			result, err := unmarshalAndValidateConfig(config)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Endpoint).To(Equal("api.test.com:8080"))
		})

		It("should accept valid URL with path and strip protocol", func() {
			config := json.RawMessage(`{
				"endpoint": "https://api.test.com/v1",
				"publicKey": "test-public-key",
				"privateKey": "test-private-key",
				"accountID": "test-account-id"
			}`)
			result, err := unmarshalAndValidateConfig(config)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Endpoint).To(Equal("api.test.com/v1"))
		})

		It("should reject empty endpoint", func() {
			config := json.RawMessage(`{
				"endpoint": "",
				"publicKey": "test-public-key",
				"privateKey": "test-private-key",
				"accountID": "test-account-id"
			}`)
			_, err := unmarshalAndValidateConfig(config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Endpoint"))
		})

		It("should reject malformed JSON", func() {
			config := json.RawMessage(`{invalid json}`)
			_, err := unmarshalAndValidateConfig(config)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("install", func() {
		It("should return valid install response", func(ctx SpecContext) {
			req := models.InstallRequest{}
			res, err := plg.Install(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Workflow).To(Equal(Workflow()))
		})
	})

	Context("uninstall", func() {
		It("should return valid uninstall response", func(ctx SpecContext) {
			req := models.UninstallRequest{ConnectorID: "test"}
			res, err := plg.Uninstall(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(models.UninstallResponse{}))
		})
	})

	Context("methods requiring installation", func() {
		It("FetchNextAccounts should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{}
			_, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("FetchNextBalances should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{}
			_, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("FetchNextExternalAccounts should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{}
			_, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("FetchNextPayments should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{}
			_, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("FetchNextOthers should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{}
			_, err := plg.FetchNextOthers(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("CreateBankAccount should fail when called before install", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{}
			_, err := plg.CreateBankAccount(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})

		It("CreateTransfer should fail when called before install", func(ctx SpecContext) {
			req := models.CreateTransferRequest{}
			_, err := plg.CreateTransfer(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})

		It("ReverseTransfer should fail when called before install", func(ctx SpecContext) {
			req := models.ReverseTransferRequest{}
			_, err := plg.ReverseTransfer(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})

		It("PollTransferStatus should fail when called before install", func(ctx SpecContext) {
			req := models.PollTransferStatusRequest{}
			_, err := plg.PollTransferStatus(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})

		It("CreatePayout should fail when called before install", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{}
			_, err := plg.CreatePayout(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("ReversePayout should fail when called before install", func(ctx SpecContext) {
			req := models.ReversePayoutRequest{}
			_, err := plg.ReversePayout(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})

		It("PollPayoutStatus should fail when called before install", func(ctx SpecContext) {
			req := models.PollPayoutStatusRequest{}
			_, err := plg.PollPayoutStatus(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})

		It("CreateWebhooks should fail when called before install", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{}
			_, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})

		It("TranslateWebhook should fail when called before install", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{}
			_, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("With installed client", func() {
		var (
			mockCtrl   *gomock.Controller
			mockClient *client.MockClient
			plg        *Plugin
		)

		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockClient = client.NewMockClient(mockCtrl)
			plg = &Plugin{
				name:   ProviderName,
				logger: logger,
				client: mockClient,
			}
		})

		AfterEach(func() {
			mockCtrl.Finish()
		})

		It("should call createPayout when CreatePayout is called", func(ctx SpecContext) {
			// Mock the expected behavior
			mockClient.EXPECT().InitiatePayout(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&moov.Transfer{TransferID: "test-reference"}, nil)

			// Test request with valid data
			req := models.CreatePayoutRequest{
				PaymentInitiation: models.PSPPaymentInitiation{
					SourceAccount: &models.PSPAccount{
						Reference: "source-ref",
						Metadata: map[string]string{
							client.MoovAccountIDMetadataKey: "source-account-id",
						},
					},
					DestinationAccount: &models.PSPAccount{
						Reference: "dest-ref",
						Metadata: map[string]string{
							client.MoovAccountIDMetadataKey: "dest-account-id",
						},
					},
					Amount: big.NewInt(1000),
					Asset:  "USD/2",
					Metadata: map[string]string{
						client.MoovSourcePaymentMethodIDMetadataKey:      "source-pm-id",
						client.MoovDestinationPaymentMethodIDMetadataKey: "dest-pm-id",
						client.MoovPaymentTypeMetadataKey:                "ach",
					},
				},
			}

			// Call the method
			resp, err := plg.CreatePayout(ctx, req)

			// Verify the response
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Payment).NotTo(BeNil())
		})
	})

	Context("when client is nil", func() {
		BeforeEach(func() {
			plg.client = nil
		})

		It("should return ErrNotYetInstalled for FetchNextAccounts", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{}
			resp, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(Equal(plugins.ErrNotYetInstalled))
			Expect(resp.Accounts).To(HaveLen(0))
		})

		It("should return ErrNotYetInstalled for FetchNextBalances", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{}
			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(Equal(plugins.ErrNotYetInstalled))
			Expect(resp.Balances).To(HaveLen(0))
		})

		It("should return ErrNotYetInstalled for FetchNextExternalAccounts", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{}
			resp, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(Equal(plugins.ErrNotYetInstalled))
			Expect(resp.ExternalAccounts).To(HaveLen(0))
		})

		It("should return ErrNotYetInstalled for FetchNextPayments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{}
			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(Equal(plugins.ErrNotYetInstalled))
			Expect(resp.Payments).To(HaveLen(0))
		})

		It("should return ErrNotYetInstalled for FetchNextOthers", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{}
			resp, err := plg.FetchNextOthers(ctx, req)
			Expect(err).To(Equal(plugins.ErrNotYetInstalled))
			Expect(resp.Others).To(HaveLen(0))
		})

		It("should return ErrNotYetInstalled for CreatePayout", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{}
			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).To(Equal(plugins.ErrNotYetInstalled))
			Expect(resp.Payment).To(BeNil())
		})

		It("should return ErrNotYetInstalled for VerifyWebhook", func(ctx SpecContext) {
			req := models.VerifyWebhookRequest{}
			resp, err := plg.VerifyWebhook(ctx, req)
			Expect(err).To(Equal(plugins.ErrNotYetInstalled))
			Expect(resp.WebhookIdempotencyKey).To(BeNil())
		})
	})

	It("should have the correct capabilities", func() {
		// The actual capabilities from the package
		expectedCapabilities := []models.Capability{
			models.CAPABILITY_FETCH_OTHERS,
			models.CAPABILITY_FETCH_ACCOUNTS,
			models.CAPABILITY_FETCH_BALANCES,
			models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS,
			models.CAPABILITY_FETCH_PAYMENTS,
			models.CAPABILITY_CREATE_PAYOUT,
		}

		// Check that the length is the same
		Expect(capabilities).To(HaveLen(len(expectedCapabilities)))

		// Check that all expected capabilities are present (order-independent)
		for _, cap := range expectedCapabilities {
			Expect(capabilities).To(ContainElement(cap))
		}
	})
})
