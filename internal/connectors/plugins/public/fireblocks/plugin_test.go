package fireblocks

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"testing"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/fireblocks/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Fireblocks Plugin Suite")
}

var _ = Describe("Fireblocks Plugin", func() {
	var (
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		pemKey string
	)

	BeforeEach(func() {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		Expect(err).To(BeNil())
		pemKey = string(pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		}))
	})

	Context("config", func() {
		It("reports validation errors when apiKey is missing", func(ctx SpecContext) {
			payload, err := json.Marshal(map[string]string{"privateKey": pemKey})
			Expect(err).To(BeNil())
			_, err = New("fireblocks", logger, payload)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("APIKey"))
		})

		It("reports validation errors when privateKey is missing", func(ctx SpecContext) {
			payload, err := json.Marshal(map[string]string{"apiKey": "test"})
			Expect(err).To(BeNil())
			_, err = New("fireblocks", logger, payload)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("PrivateKey"))
		})

		It("rejects malformed private keys", func(ctx SpecContext) {
			payload, err := json.Marshal(map[string]string{
				"apiKey":     "test",
				"privateKey": "bad",
			})
			Expect(err).To(BeNil())
			_, err = New("fireblocks", logger, payload)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("invalid private key"))
		})

		It("defaults base URL when empty", func(ctx SpecContext) {
			payload, err := json.Marshal(map[string]string{
				"apiKey":     "test",
				"privateKey": pemKey,
			})
			Expect(err).To(BeNil())
			config, err := unmarshalAndValidateConfig(payload)
			Expect(err).To(BeNil())
			Expect(config.BaseURL).To(Equal(DefaultBaseURL))
		})
	})

	Context("install", func() {
		var (
			ctrl *gomock.Controller
			m    *client.MockClient
			plg  *Plugin
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{
				Plugin: plugins.NewBasePlugin(),
				client: m,
				logger: logger,
			}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("loads asset decimals and returns workflow", func(ctx SpecContext) {
			m.EXPECT().ListAssets(gomock.Any()).Return([]client.Asset{
				{LegacyID: "", Decimals: 2},
				{LegacyID: "BTC", Onchain: &client.AssetOnchain{Decimals: 8}},
				{LegacyID: "USD", Decimals: 2},
				{LegacyID: "NO_DECIMALS"},
				{LegacyID: "NEG", Onchain: &client.AssetOnchain{Decimals: -1}},
			}, nil)

			res, err := plg.Install(ctx, models.InstallRequest{})
			Expect(err).To(BeNil())
			Expect(res.Workflow).To(Equal(workflow()))
			Expect(plg.assetDecimals).To(HaveLen(2))
			Expect(plg.assetDecimals["BTC"]).To(Equal(8))
			Expect(plg.assetDecimals["USD"]).To(Equal(2))
		})
	})
})
