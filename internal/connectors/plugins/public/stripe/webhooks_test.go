package stripe

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/public/stripe/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/webhook"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Stripe Plugin Webhooks", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  models.Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m, logger: logging.Testing()}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Create Webhooks", func() {
		It("returns client errors", func(ctx SpecContext) {
			expectedErr := errors.New("webhook err")
			req := models.CreateWebhooksRequest{WebhookBaseUrl: "http://example.com"}
			m.EXPECT().CreateWebhookEndpoints(gomock.Any(), req.WebhookBaseUrl).Return(nil, expectedErr)
			_, err := plg.CreateWebhooks(ctx, req)
			Expect(err).NotTo(BeNil())
			Expect(err).To(Equal(expectedErr))
		})

		It("returns list of webhooks created", func(ctx SpecContext) {
			rootAccountID := "rooootAcc"
			endpoints := []*stripe.WebhookEndpoint{
				{
					ID:            "id1",
					URL:           "http://example.com/endpoint1",
					Secret:        "seeeecreeet",
					EnabledEvents: []string{"some.event"},
				},
				{
					ID:            "id2",
					URL:           "http://example.com/connect_endpoint2",
					Secret:        "seeeecreeet2",
					EnabledEvents: []string{"some.event", "some.event2"},
				},
			}
			req := models.CreateWebhooksRequest{WebhookBaseUrl: "http://example.com"}
			m.EXPECT().GetRootAccountID().MaxTimes(1).Return(rootAccountID)
			m.EXPECT().CreateWebhookEndpoints(gomock.Any(), req.WebhookBaseUrl).Return(endpoints, nil)
			result, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(BeNil())
			Expect(result.Others).To(HaveLen(len(endpoints)))
			Expect(result.Configs).To(HaveLen(len(endpoints)))

			configs := result.Configs
			Expect(configs[0].Name).To(Equal("id1"))
			Expect(configs[0].URLPath).To(Equal("/endpoint1"))
			Expect(configs[0].Metadata).To(Equal(map[string]string{
				"secret":                   endpoints[0].Secret,
				webhookRelatedAccountIDKey: rootAccountID,
				"enabled_events":           "some.event",
			}))
			Expect(configs[1].Name).To(Equal("id2"))
			Expect(configs[1].URLPath).To(Equal("/connect_endpoint2"))
			Expect(configs[1].Metadata).To(Equal(map[string]string{
				"secret":         endpoints[1].Secret,
				"enabled_events": "some.event,some.event2",
			}))

			Expect(result.Others[0].ID).To(Equal("id1"))
			Expect(result.Others[1].ID).To(Equal("id2"))
		})
	})

	Context("Translate Webhooks", func() {
		var (
			rootAccount string
			secret      string
			balance     *stripe.Balance
			pspWebhook  = func(secretVal string, payload []byte) models.PSPWebhook {
				timestamp := time.Now()
				rawSignature := webhook.ComputeSignature(timestamp, payload, secretVal)
				signature := fmt.Sprintf("t=%d,v1=%s", timestamp.Unix(), hex.EncodeToString(rawSignature))
				return models.PSPWebhook{
					Headers: map[string][]string{"Stripe-Signature": []string{string(signature)}},
					Body:    payload,
				}
			}
		)

		BeforeEach(func() {
			rootAccount = "acc_rooooot"
			secret = "the_secret"
			balance = &stripe.Balance{Available: []*stripe.Amount{
				{
					Amount:   1345,
					Currency: stripe.CurrencyAUD,
				},
			}}
		})

		It("fails when signature is signed with a different secret", func(ctx SpecContext) {
			rootAccount := "acc_rooooot"
			e := &stripe.Event{
				APIVersion: stripe.APIVersion,
				Type:       stripe.EventTypeBalanceAvailable,
				Data:       &stripe.EventData{Raw: json.RawMessage("{}")},
			}
			payload, err := json.Marshal(e)
			Expect(err).To(BeNil())

			req := models.TranslateWebhookRequest{
				Name:    "some_name",
				Webhook: pspWebhook(secret, payload),
				Config: &models.WebhookConfig{
					Metadata: map[string]string{
						"secret":                   "differentSecretExpected",
						webhookRelatedAccountIDKey: rootAccount,
					},
				},
			}
			_, err = plg.TranslateWebhook(ctx, req)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(models.ErrWebhookVerification))
		})

		It("fails when a balance.available webhook is missing account info", func(ctx SpecContext) {
			innerPayload, err := json.Marshal(balance)
			Expect(err).To(BeNil())

			e := &stripe.Event{
				Created:    time.Now().Unix(),
				APIVersion: stripe.APIVersion,
				Type:       stripe.EventTypeBalanceAvailable,
				Data:       &stripe.EventData{Raw: json.RawMessage(innerPayload)},
			}
			payload, err := json.Marshal(e)
			Expect(err).To(BeNil())

			req := models.TranslateWebhookRequest{
				Name:    "some_name",
				Webhook: pspWebhook(secret, payload),
				Config: &models.WebhookConfig{
					Metadata: map[string]string{
						"secret": secret,
					},
				},
			}
			_, err = plg.TranslateWebhook(ctx, req)
			Expect(err).NotTo(BeNil())
		})

		It("translates a balance.available webhook for the root account", func(ctx SpecContext) {
			innerPayload, err := json.Marshal(balance)
			Expect(err).To(BeNil())

			e := &stripe.Event{
				Created:    time.Now().Unix(),
				APIVersion: stripe.APIVersion,
				Type:       stripe.EventTypeBalanceAvailable,
				Data:       &stripe.EventData{Raw: json.RawMessage(innerPayload)},
			}
			payload, err := json.Marshal(e)
			Expect(err).To(BeNil())

			req := models.TranslateWebhookRequest{
				Name:    "some_name",
				Webhook: pspWebhook(secret, payload),
				Config: &models.WebhookConfig{
					Metadata: map[string]string{
						"secret":                   secret,
						webhookRelatedAccountIDKey: rootAccount,
					},
				},
			}
			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(res.Responses[0].Balance).NotTo(BeNil())
			Expect(res.Responses[0].Balance.AccountReference).To(Equal(rootAccount))
			Expect(res.Responses[0].Balance.CreatedAt.Unix()).To(Equal(e.Created))
			Expect(res.Responses[0].Balance.Asset).To(Equal("AUD/2"))
			Expect(res.Responses[0].Balance.Amount).To(Equal(big.NewInt(balance.Available[0].Amount)))
		})

		It("translates a balance.available webhook from a Stripe connect account", func(ctx SpecContext) {
			innerPayload, err := json.Marshal(balance)
			Expect(err).To(BeNil())

			e := &stripe.Event{
				Created:    time.Now().Unix(),
				Account:    "acc_otheraccount",
				APIVersion: stripe.APIVersion,
				Type:       stripe.EventTypeBalanceAvailable,
				Data:       &stripe.EventData{Raw: json.RawMessage(innerPayload)},
			}
			payload, err := json.Marshal(e)
			Expect(err).To(BeNil())

			req := models.TranslateWebhookRequest{
				Name:    "some_name",
				Webhook: pspWebhook(secret, payload),
				Config: &models.WebhookConfig{
					Metadata: map[string]string{
						"secret":                   secret,
						webhookRelatedAccountIDKey: rootAccount,
					},
				},
			}
			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(res.Responses[0].Balance).NotTo(BeNil())
			Expect(res.Responses[0].Balance.AccountReference).To(Equal(e.Account))
			Expect(res.Responses[0].Balance.CreatedAt.Unix()).To(Equal(e.Created))
			Expect(res.Responses[0].Balance.Asset).To(Equal("AUD/2"))
			Expect(res.Responses[0].Balance.Amount).To(Equal(big.NewInt(balance.Available[0].Amount)))
		})
	})
})
