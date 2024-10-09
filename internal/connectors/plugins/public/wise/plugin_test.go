package wise

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"testing"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Wise Plugin Suite")
}

var _ = Describe("Wise Plugin", func() {
	var (
		plg    *Plugin
		block  *pem.Block
		pemKey *bytes.Buffer
	)

	BeforeEach(func() {
		plg = &Plugin{}

		privatekey, err := rsa.GenerateKey(rand.Reader, 2048)
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
			req := models.InstallRequest{Config: json.RawMessage(`{}`)}
			_, err := plg.Install(context.Background(), req)
			Expect(err).To(MatchError(ContainSubstring("config")))
		})
		It("rejects malformed pem keys", func(ctx SpecContext) {
			config := json.RawMessage(`{"apiKey":"dummy","webhookPublicKey":"badKey"}`)
			req := models.InstallRequest{Config: config}
			_, err := plg.Install(context.Background(), req)
			Expect(err).To(MatchError(ContainSubstring("public key")))
		})
		It("returns valid install response", func(ctx SpecContext) {
			config := &Config{
				APIKey:           "key",
				WebhookPublicKey: pemKey.String(),
			}
			configJson, err := json.Marshal(config)
			req := models.InstallRequest{Config: configJson}
			res, err := plg.Install(context.Background(), req)
			Expect(err).To(BeNil())
			Expect(len(res.Capabilities) > 0).To(BeTrue())
			Expect(len(res.Workflow) > 0).To(BeTrue())
			Expect(res.Workflow[0].Name).To(Equal("fetch_profiles"))
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
	})
})
