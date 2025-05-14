package wise

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/wise/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Wise Plugin Suite")
}

var _ = Describe("Wise Plugin", func() {
	var (
		plg        *Plugin
		block      *pem.Block
		pemKey     *bytes.Buffer
		privatekey *rsa.PrivateKey
		logger     = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		plg = &Plugin{
			Plugin: plugins.NewBasePlugin(),
		}

		plg.supportedWebhooks = map[string]supportedWebhook{
			"transfer_state_changed": {
				triggerOn: "transfers#state-change",
				urlPath:   "/transferstatechanged",
				fn:        plg.translateTransferStateChangedWebhook,
				version:   "2.0.0",
			},
			"balance_update": {
				triggerOn: "balances#update",
				urlPath:   "/balanceupdate",
				fn:        plg.translateBalanceUpdateWebhook,
				version:   "2.2.0",
			},
		}

		var err error
		privatekey, err = rsa.GenerateKey(rand.Reader, 2048)
		Expect(err).To(BeNil())
		publickey := &privatekey.PublicKey
		publicKeyBytes, err := x509.MarshalPKIXPublicKey(publickey)
		Expect(err).To(BeNil())
		block = &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: publicKeyBytes,
		}
		pemKey = bytes.NewBufferString("")

		err = pem.Encode(pemKey, block)
		Expect(err).To(BeNil())
	})

	Context("install", func() {
		It("reports validation errors in the config", func(ctx SpecContext) {
			_, err := New("wise", logger, json.RawMessage(`{}`))
			Expect(err).To(MatchError(ContainSubstring("validation")))
		})
		It("rejects malformed pem keys", func(ctx SpecContext) {
			config := json.RawMessage(`{"apiKey":"dummy","webhookPublicKey":"badKey"}`)
			_, err := New("wise", logger, config)
			Expect(err).To(MatchError(ContainSubstring("public key")))
		})
		It("returns valid install response", func(ctx SpecContext) {
			config := &Config{
				APIKey:           "key",
				WebhookPublicKey: pemKey.String(),
			}
			configJson, err := json.Marshal(config)
			Expect(err).To(BeNil())
			_, err = New("wise", logger, configJson)
			Expect(err).To(BeNil())
			req := models.InstallRequest{}
			res, err := plg.Install(context.Background(), req)
			Expect(err).To(BeNil())
			Expect(len(res.Workflow) > 0).To(BeTrue())
			Expect(res.Workflow[0].Name).To(Equal("fetch_profiles"))
		})
	})

	Context("verifying webhook", func() {
		var (
			body      []byte
			signature []byte
			m         *client.MockClient
		)

		BeforeEach(func() {
			config := &Config{
				APIKey:           "key",
				WebhookPublicKey: pemKey.String(),
			}
			configJson, err := json.Marshal(config)
			Expect(err).To(BeNil())
			plg, err = New("wise", logger, configJson)
			Expect(err).To(BeNil())

			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.SetClient(m)

			body = bytes.NewBufferString("body content").Bytes()
			hash := sha256.New()
			hash.Write(body)
			digest := hash.Sum(nil)

			signature, err = rsa.SignPKCS1v15(rand.Reader, privatekey, crypto.SHA256, digest)
			Expect(err).To(BeNil())
		})

		It("it fails when X-Delivery-ID header missing", func(ctx SpecContext) {
			req := models.VerifyWebhookRequest{}
			_, err := plg.VerifyWebhook(context.Background(), req)
			Expect(err).To(MatchError(ErrWebhookHeaderXDeliveryIDMissing))
		})

		It("it fails when X-Signature-Sha256 header missing", func(ctx SpecContext) {
			req := models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						HeadersDeliveryID: {"delivery-id"},
					},
				},
			}
			_, err := plg.VerifyWebhook(context.Background(), req)
			Expect(err).To(MatchError(ErrWebhookHeaderXSignatureMissing))
		})

		It("it can create the balance_update webhook", func(ctx SpecContext) {
			req := models.VerifyWebhookRequest{
				Config: &models.WebhookConfig{Name: "balance_update"},
				Webhook: models.PSPWebhook{
					Body: body,
					Headers: map[string][]string{
						HeadersDeliveryID: {"delivery-id"},
						HeadersSignature:  {base64.StdEncoding.EncodeToString(signature)},
					},
				},
			}

			res, err := plg.VerifyWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.WebhookIdempotencyKey).To(Equal(&req.Webhook.Headers[HeadersDeliveryID][0]))
		})
	})

	Context("translate webhook", func() {
		var (
			body      []byte
			signature []byte
			m         *client.MockClient
		)

		BeforeEach(func() {
			config := &Config{
				APIKey:           "key",
				WebhookPublicKey: pemKey.String(),
			}
			configJson, err := json.Marshal(config)
			Expect(err).To(BeNil())
			plg, err = New("wise", logger, configJson)
			Expect(err).To(BeNil())

			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.SetClient(m)

			body = bytes.NewBufferString("body content").Bytes()
			hash := sha256.New()
			hash.Write(body)
			digest := hash.Sum(nil)

			signature, err = rsa.SignPKCS1v15(rand.Reader, privatekey, crypto.SHA256, digest)
			Expect(err).To(BeNil())
		})

		It("it fails when unknown webhook name in request", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "unknown",
				Webhook: models.PSPWebhook{
					Body: body,
					Headers: map[string][]string{
						HeadersDeliveryID: {"delivery-id"},
						HeadersSignature:  {base64.StdEncoding.EncodeToString(signature)},
					},
				},
			}

			_, err := plg.TranslateWebhook(context.Background(), req)
			Expect(err).To(MatchError(ErrWebhookNameUnknown))
		})

		It("it can create the transfer_state_changed webhook", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "transfer_state_changed",
				Webhook: models.PSPWebhook{
					Body: body,
					Headers: map[string][]string{
						HeadersDeliveryID: {"delivery-id"},
						HeadersSignature:  {base64.StdEncoding.EncodeToString(signature)},
					},
				},
			}
			transfer := client.Transfer{ID: 1, Reference: "ref1", TargetValue: json.Number("25"), TargetCurrency: "EUR"}
			m.EXPECT().TranslateTransferStateChangedWebhook(gomock.Any(), body).Return(transfer, nil)

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(res.Responses[0].Payment).NotTo(BeNil())
			Expect(res.Responses[0].Payment.Reference).To(Equal(fmt.Sprint(transfer.ID)))
		})

		It("it can create the balance_update webhook", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "balance_update",
				Webhook: models.PSPWebhook{
					Body: body,
					Headers: map[string][]string{
						HeadersDeliveryID: {"delivery-id"},
						HeadersSignature:  {base64.StdEncoding.EncodeToString(signature)},
					},
				},
			}
			balance := client.BalanceUpdateWebhookPayload{
				Data: client.BalanceUpdateWebhookData{
					TransferReference: "trx",
					OccurredAt:        time.Now().Format(time.RFC3339),
					Currency:          "USD",
					Amount:            json.Number("43"),
				},
			}
			m.EXPECT().TranslateBalanceUpdateWebhook(gomock.Any(), body).Return(balance, nil)

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(res.Responses[0].Payment).NotTo(BeNil())
			Expect(res.Responses[0].Payment.Reference).To(Equal(balance.Data.TransferReference))
		})
	})

	Context("calling functions on uninstalled plugins", func() {
		It("returns valid uninstall response", func(ctx SpecContext) {
			req := models.UninstallRequest{ConnectorID: "dummyID"}
			_, err := plg.Uninstall(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("fails when fetch next accounts is called before install", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State: json.RawMessage(`{}`),
			}
			_, err := plg.FetchNextAccounts(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		It("fails when fetch next balances is called before install", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				State: json.RawMessage(`{}`),
			}
			_, err := plg.FetchNextBalances(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		It("fails when fetch next others is called before install", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{
				State: json.RawMessage(`{}`),
			}
			_, err := plg.FetchNextOthers(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		It("fails when fetch next external accounts is called before install", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{
				State: json.RawMessage(`{}`),
			}
			_, err := plg.FetchNextExternalAccounts(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		It("fails when create webhook is called before install", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{}
			_, err := plg.CreateWebhooks(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		It("fails when create transfer is called before install", func(ctx SpecContext) {
			req := models.CreateTransferRequest{}
			_, err := plg.CreateTransfer(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		It("fails when create transfer is called before install", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{}
			_, err := plg.CreatePayout(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})
})
