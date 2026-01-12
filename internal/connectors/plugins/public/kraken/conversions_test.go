package kraken

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/public/kraken/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Kraken Plugin Conversions", func() {
	var (
		ctrl   *gomock.Controller
		m      *client.MockClient
		plg    *Plugin
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{
			client: m,
			logger: logger,
			config: Config{},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetch next conversions", func() {
		It("returns empty list as Kraken has no dedicated conversions endpoint", func(ctx SpecContext) {
			req := models.FetchNextConversionsRequest{
				State:    json.RawMessage(`{}`),
				PageSize: 100,
			}

			res, err := plg.FetchNextConversions(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.HasMore).To(BeFalse())
			Expect(res.Conversions).To(BeEmpty())
		})
	})

	Context("create conversion", func() {
		It("creates a conversion successfully", func(ctx SpecContext) {
			req := models.CreateConversionRequest{
				Conversion: models.PSPConversion{
					WalletID:     "wallet-123",
					SourceAsset:  "BTC/8",
					TargetAsset:  "USD/2",
					SourceAmount: big.NewInt(100000000), // 1 BTC
					Reference:    "test-ref",
				},
			}

			m.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).Return(
				&client.CreateOrderResponse{
					TxID: []string{"OXXXXX-XXXXX-XXXXXX"},
				},
				nil,
			)

			res, err := plg.CreateConversion(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Conversion).NotTo(BeNil())
			Expect(res.Conversion.Reference).To(Equal("OXXXXX-XXXXX-XXXXXX"))
			Expect(res.Conversion.Status).To(Equal(models.CONVERSION_STATUS_PENDING))
		})

		It("returns error when no transaction ID is returned", func(ctx SpecContext) {
			req := models.CreateConversionRequest{
				Conversion: models.PSPConversion{
					WalletID:     "wallet-123",
					SourceAsset:  "BTC/8",
					TargetAsset:  "USD/2",
					SourceAmount: big.NewInt(100000000),
				},
			}

			m.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).Return(
				&client.CreateOrderResponse{
					TxID: []string{},
				},
				nil,
			)

			_, err := plg.CreateConversion(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no transaction ID returned from Kraken"))
		})

		It("returns error on client failure", func(ctx SpecContext) {
			req := models.CreateConversionRequest{
				Conversion: models.PSPConversion{
					WalletID:     "wallet-123",
					SourceAsset:  "BTC/8",
					TargetAsset:  "USD/2",
					SourceAmount: big.NewInt(100000000),
				},
			}

			m.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).Return(
				nil,
				fmt.Errorf("client error"),
			)

			_, err := plg.CreateConversion(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to create conversion"))
		})

		It("uses first transaction ID when multiple are returned", func(ctx SpecContext) {
			req := models.CreateConversionRequest{
				Conversion: models.PSPConversion{
					WalletID:     "wallet-123",
					SourceAsset:  "ETH/18",
					TargetAsset:  "USD/2",
					SourceAmount: big.NewInt(1000000000000000000), // 1 ETH
				},
			}

			m.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).Return(
				&client.CreateOrderResponse{
					TxID: []string{"FIRST-TX-ID", "SECOND-TX-ID"},
				},
				nil,
			)

			res, err := plg.CreateConversion(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Conversion).NotTo(BeNil())
			Expect(res.Conversion.Reference).To(Equal("FIRST-TX-ID"))
		})
	})

	Context("helper functions", func() {
		It("strips asset precision correctly", func() {
			Expect(stripAssetPrecision("BTC/8")).To(Equal("BTC"))
			Expect(stripAssetPrecision("USD/2")).To(Equal("USD"))
			Expect(stripAssetPrecision("ETH")).To(Equal("ETH"))
		})
	})
})
