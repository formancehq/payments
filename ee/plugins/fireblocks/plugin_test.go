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
	"github.com/formancehq/payments/ee/plugins/fireblocks/client"
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

		It("defaults endpoint when empty", func(ctx SpecContext) {
			payload, err := json.Marshal(map[string]string{
				"apiKey":     "test",
				"privateKey": pemKey,
			})
			Expect(err).To(BeNil())
			config, err := unmarshalAndValidateConfig(payload)
			Expect(err).To(BeNil())
			Expect(config.Endpoint).To(Equal(DefaultEndpoint))
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

		It("loads assets keyed by uppercased legacyId and returns workflow", func(ctx SpecContext) {
			decimals2 := 2
			decimals0 := 0
			m.EXPECT().ListBlockchains(gomock.Any()).Return([]client.Blockchain{
				{ID: "chain-eth", Onchain: &client.BlockchainOnchain{Test: false}},
				{ID: "chain-eth-sepolia", Onchain: &client.BlockchainOnchain{Test: true}},
			}, nil)
			m.EXPECT().ListAssets(gomock.Any()).Return([]client.Asset{
				{LegacyID: "", DisplaySymbol: "X", AssetClass: client.AssetClassFiat, Decimals: &decimals2}, // skipped: no legacyId
				{LegacyID: "BTC", DisplaySymbol: "BTC", AssetClass: client.AssetClassNative, BlockchainID: "chain-eth", Onchain: &client.AssetOnchain{Decimals: 8}},
				{LegacyID: "USD", DisplaySymbol: "USD", AssetClass: client.AssetClassFiat, Decimals: &decimals2},
				{LegacyID: "JPY", DisplaySymbol: "JPY", AssetClass: client.AssetClassFiat, Decimals: &decimals0},
				{LegacyID: "ETH_TEST5", DisplaySymbol: "ETH", AssetClass: client.AssetClassNative, BlockchainID: "chain-eth-sepolia", Onchain: &client.AssetOnchain{Decimals: 18}},
				{LegacyID: "NEG", DisplaySymbol: "NEG", AssetClass: client.AssetClassNative, Onchain: &client.AssetOnchain{Decimals: -1}}, // skipped: negative
			}, nil)

			res, err := plg.Install(ctx, models.InstallRequest{})
			Expect(err).To(BeNil())
			Expect(res.Workflow).To(Equal(workflow()))
			Expect(plg.assets).To(HaveLen(4))
			Expect(plg.assets["BTC"].Asset).To(Equal("BTC/8"))
			Expect(plg.assets["USD"].Asset).To(Equal("USD/2"))
			Expect(plg.assets["JPY"].Asset).To(Equal("JPY"))
			Expect(plg.assets["ETH_TEST5"].Asset).To(Equal("ETH_TEST/18"))
			Expect(plg.assets["ETH_TEST5"].Metadata).To(HaveKeyWithValue(MetadataPrefix+"testnet", "true"))
		})
	})
})
