package plaid

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/formancehq/payments/ce/plugins/plaid/client"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/golang-jwt/jwt/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/plaid/plaid-go/v34/plaid"
	gomock "go.uber.org/mock/gomock"
)

func encodePlaidCoord(b []byte) string {
	if len(b) < 32 {
		padded := make([]byte, 32)
		copy(padded[32-len(b):], b)
		b = padded
	}
	return strings.TrimRight(base64.URLEncoding.EncodeToString(b), "=")
}

func signPlaidWebhookJWT(priv *ecdsa.PrivateKey, kid string, claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = kid
	s, err := token.SignedString(priv)
	Expect(err).To(BeNil())
	return s
}

func plaidJWK(priv *ecdsa.PrivateKey) *plaid.JWKPublicKey {
	key := plaid.NewJWKPublicKeyWithDefaults()
	key.SetX(encodePlaidCoord(priv.PublicKey.X.Bytes()))
	key.SetY(encodePlaidCoord(priv.PublicKey.Y.Bytes()))
	return key
}

var _ = Describe("Plaid *Plugin Webhooks", func() {
	Context("create webhooks", func() {
		var (
			ctrl *gomock.Controller
			plg  models.Plugin
			m    *client.MockClient
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			p := &Plugin{
				client: m,
			}

			p.supportedWebhooks = map[string]supportedWebhook{
				"all": {
					urlPath: "/all",
					fn:      p.handleAllWebhook,
				},
			}

			plg = p
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should create webhooks successfully", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{}

			resp, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Configs).To(HaveLen(1))
			Expect(resp.Configs[0].Name).To(Equal("all"))
			Expect(resp.Configs[0].URLPath).To(Equal("/all"))
		})
	})

	Context("verify webhook", func() {
		var (
			ctrl *gomock.Controller
			plg  models.Plugin
			m    *client.MockClient
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{client: m}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should return an error - missing Plaid-Verification header", func(ctx SpecContext) {
			req := models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{},
				},
			}

			resp, err := plg.VerifyWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("invalid token"))
			Expect(resp).To(Equal(models.VerifyWebhookResponse{}))
		})

		It("should return an error - multiple Plaid-Verification headers", func(ctx SpecContext) {
			req := models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Plaid-Verification": {"token1", "token2"},
					},
				},
			}

			resp, err := plg.VerifyWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("invalid token"))
			Expect(resp).To(Equal(models.VerifyWebhookResponse{}))
		})

		It("should accept a signed JWT whose iat is fresh and body hash matches", func(ctx SpecContext) {
			priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			Expect(err).To(BeNil())
			body := []byte(`{"webhook_type":"TRANSACTIONS","webhook_code":"SYNC_UPDATES_AVAILABLE"}`)
			sum := sha256.Sum256(body)
			token := signPlaidWebhookJWT(priv, "kid-1", jwt.MapClaims{
				"iat":                 time.Now().Unix(),
				"request_body_sha256": hex.EncodeToString(sum[:]),
			})
			m.EXPECT().GetWebhookVerificationKey(ctx, "kid-1").Return(plaidJWK(priv), nil)

			resp, err := plg.VerifyWebhook(ctx, models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{"Plaid-Verification": {token}},
					Body:    body,
				},
			})
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.VerifyWebhookResponse{}))
		})

		It("should reject a signed JWT when the body hash does not match", func(ctx SpecContext) {
			priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			Expect(err).To(BeNil())
			body := []byte(`{"webhook_type":"ITEM","webhook_code":"ERROR"}`)
			other := sha256.Sum256([]byte(`{"forged":true}`))
			token := signPlaidWebhookJWT(priv, "kid-1", jwt.MapClaims{
				"iat":                 time.Now().Unix(),
				"request_body_sha256": hex.EncodeToString(other[:]),
			})
			m.EXPECT().GetWebhookVerificationKey(ctx, "kid-1").Return(plaidJWK(priv), nil)

			_, err = plg.VerifyWebhook(ctx, models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{"Plaid-Verification": {token}},
					Body:    body,
				},
			})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("request_body_sha256 mismatch"))
		})

		It("should reject a signed JWT with no request_body_sha256", func(ctx SpecContext) {
			priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			Expect(err).To(BeNil())
			token := signPlaidWebhookJWT(priv, "kid-1", jwt.MapClaims{
				"iat": time.Now().Unix(),
			})
			m.EXPECT().GetWebhookVerificationKey(ctx, "kid-1").Return(plaidJWK(priv), nil)

			_, err = plg.VerifyWebhook(ctx, models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{"Plaid-Verification": {token}},
					Body:    []byte(`{}`),
				},
			})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing request_body_sha256"))
		})

		It("should reject a signed JWT whose iat is older than 5 minutes", func(ctx SpecContext) {
			priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			Expect(err).To(BeNil())
			body := []byte(`{"webhook_type":"TRANSACTIONS"}`)
			sum := sha256.Sum256(body)
			token := signPlaidWebhookJWT(priv, "kid-1", jwt.MapClaims{
				"iat":                 time.Now().Add(-6 * time.Minute).Unix(),
				"request_body_sha256": hex.EncodeToString(sum[:]),
			})
			m.EXPECT().GetWebhookVerificationKey(ctx, "kid-1").Return(plaidJWK(priv), nil)

			_, err = plg.VerifyWebhook(ctx, models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{"Plaid-Verification": {token}},
					Body:    body,
				},
			})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("older than 5 minutes"))
		})
	})

	Context("translate webhook", func() {
		var (
			ctrl *gomock.Controller
			plg  models.Plugin
			m    *client.MockClient
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			p := &Plugin{client: m}
			p.supportedWebhooks = map[string]supportedWebhook{
				"all": {
					urlPath: "/all",
					fn:      p.handleAllWebhook,
				},
			}

			plg = p
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should return an error - unsupported webhook name", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "unsupported",
				Webhook: models.PSPWebhook{
					Body: []byte(`{}`),
				},
			}

			resp, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("unsupported webhook"))
			Expect(resp).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should translate webhook successfully", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "all",
				Webhook: models.PSPWebhook{
					Body: []byte(`{"webhook_type": "TRANSACTIONS", "webhook_code": "SYNC_UPDATES_AVAILABLE"}`),
				},
			}

			// Mock the BaseWebhookTranslation method
			baseWebhook := client.BaseWebhooks{
				WebhookType: "TRANSACTIONS",
				WebhookCode: "SYNC_UPDATES_AVAILABLE",
			}

			m.EXPECT().BaseWebhookTranslation(req.Webhook.Body).Return(baseWebhook, nil)

			resp, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).ToNot(Equal(models.TranslateWebhookResponse{}))
		})

		It("should return an error - base webhook translation error", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "all",
				Webhook: models.PSPWebhook{
					Body: []byte(`invalid json`),
				},
			}

			m.EXPECT().BaseWebhookTranslation(req.Webhook.Body).Return(client.BaseWebhooks{}, errors.New("translation error"))

			resp, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("translation error"))
			Expect(resp).To(Equal(models.TranslateWebhookResponse{}))
		})
	})
})
