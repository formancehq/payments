package increase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/golang/mock/gomock"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Increase Plugin Suite")
}

var _ = Describe("Increase Plugin", func() {
	var (
		plg         *Plugin
		logger      = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		connectorID = models.ConnectorID{
			Reference: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
			Provider:  ProviderName,
		}
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("install", func() {
		It("should report errors in config - apiKey", func(ctx SpecContext) {
			config := json.RawMessage(`{"endpoint": "test"}`)
			_, err := New(connectorID, ProviderName, logger, config)
			Expect(err.Error()).To(ContainSubstring("APIKey"))
		})

		It("should report errors in config - endpoint", func(ctx SpecContext) {
			config := json.RawMessage(`{"apiKey": "test"}`)
			_, err := New(connectorID, ProviderName, logger, config)
			Expect(err.Error()).To(ContainSubstring("Endpoint"))
		})

		It("should return valid install response", func(ctx SpecContext) {
			config := json.RawMessage(`{"apiKey": "test", "endpoint": "test", "webhookSharedSecret": "secret"}`)
			plg, err := New(connectorID, ProviderName, logger, config)
			Expect(err).To(BeNil())
			req := models.InstallRequest{}
			res, err := plg.Install(ctx, req)
			Expect(err).To(BeNil())
			Expect(len(res.Workflow) > 0).To(BeTrue())
			Expect(res.Workflow).To(Equal(workflow()))
		})
	})

	Context("uninstall", func() {
		It("should return valid uninstall response", func(ctx SpecContext) {
			req := models.UninstallRequest{ConnectorID: "test"}
			_, err := plg.Uninstall(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("fetch next accounts", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		// Other tests will be in accounts_test.go
	})

	Context("fetch next balances", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		// Other tests will be in balances_test.go
	})

	Context("fetch next external accounts", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		// Other tests will be in external_accounts_test.go
	})

	Context("fetch next payments", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		// Other tests will be in payments_test.go
	})

	Context("fetch next others", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextOthers(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("create bank account", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{}
			_, err := plg.CreateBankAccount(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("create transfer", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.CreateTransferRequest{}
			_, err := plg.CreateTransfer(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		// Other tests will be in transfers_test.go
	})

	Context("reverse transfer", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.ReverseTransferRequest{}
			_, err := plg.ReverseTransfer(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("poll transfer status", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.PollTransferStatusRequest{}
			_, err := plg.PollTransferStatus(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("create payout", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{}
			_, err := plg.CreatePayout(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		// Other tests will be in payouts_test.go
	})

	Context("reverse payout", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.ReversePayoutRequest{}
			_, err := plg.ReversePayout(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("poll payout status", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.PollPayoutStatusRequest{}
			_, err := plg.PollPayoutStatus(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("verifying webhook", func() {
		var (
			body      []byte
			signature string
			ctrl      *gomock.Controller
			m         *client.MockHTTPClient
		)

		BeforeEach(func() {
			config := &Config{
				APIKey:              "key",
				Endpoint:            "https://api.increase.com",
				WebhookSharedSecret: "secret",
			}
			configJson, err := json.Marshal(config)
			Expect(err).To(BeNil())
			plg, err = New(connectorID, ProviderName, logger, configJson)
			Expect(err).To(BeNil())

			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockHTTPClient(ctrl)
			plg.client.SetHttpClient(m)

			body = bytes.NewBufferString(`{"id":"1", "associated_object_id": "2345678"}`).Bytes()
			timestamp := time.Now().UTC().Format(time.RFC3339)
			signedPayload := fmt.Sprintf("%s.%s", timestamp, string(body))
			signature, err = computeHMACSHA256(signedPayload, "secret")
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("it fails when X-Signature-Sha256 header missing", func(ctx SpecContext) {
			req := models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{},
				},
			}
			_, err := plg.VerifyWebhook(context.Background(), req)
			Expect(err).To(MatchError(client.ErrWebhookHeaderXSignatureMissing))
		})

		It("it can verify webhook successfully", func(ctx SpecContext) {
			timestamp := time.Now().UTC().Format(time.RFC3339)
			req := models.VerifyWebhookRequest{
				Config: &models.WebhookConfig{Name: "transaction.created"},
				Webhook: models.PSPWebhook{
					Body: body,
					Headers: map[string][]string{
						HeadersSignature: {fmt.Sprintf("t=%s,v1=%s", timestamp, signature)},
					},
				},
			}

			res, err := plg.VerifyWebhook(ctx, req)
			Expect(err).To(BeNil())
			id := "1"
			Expect(res.WebhookIdempotencyKey).To(Equal(&id))
		})
	})

	Context("create webhooks", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{}
			_, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("translate webhook", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{}
			_, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	It("should have the correct capabilities", func() {
		expectedCapabilities := []models.Capability{
			models.CAPABILITY_FETCH_ACCOUNTS,
			models.CAPABILITY_FETCH_BALANCES,
			models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS,
			models.CAPABILITY_FETCH_PAYMENTS,
			models.CAPABILITY_CREATE_TRANSFER,
			models.CAPABILITY_CREATE_PAYOUT,
			models.CAPABILITY_CREATE_BANK_ACCOUNT,
			models.CAPABILITY_TRANSLATE_WEBHOOKS,
			models.CAPABILITY_CREATE_WEBHOOKS,
		}

		Expect(capabilities).To(HaveLen(len(expectedCapabilities)))
		Expect(capabilities).To(Equal(expectedCapabilities))

		// Verify each capability is present
		for _, expectedCap := range expectedCapabilities {
			Expect(capabilities).To(ContainElement(expectedCap))
		}
	})
})
