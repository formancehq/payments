package bitstamp

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/public/bitstamp/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Bitstamp Plugin Conversions", func() {
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
		It("returns empty list as Bitstamp has no dedicated conversions endpoint", func(ctx SpecContext) {
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

			m.EXPECT().CreateMarketSellOrder(gomock.Any(), gomock.Any()).Return(
				&client.CreateOrderResponse{
					ID:            "123456789",
					DateTime:      "2024-01-01 00:00:00",
					Type:          "1", // sell
					Price:         "42000.00",
					Amount:        "1.00000000",
					ClientOrderID: "test-ref",
				},
				nil,
			)

			res, err := plg.CreateConversion(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Conversion).NotTo(BeNil())
			Expect(res.Conversion.Reference).To(Equal("123456789"))
			Expect(res.Conversion.Status).To(Equal(models.CONVERSION_STATUS_COMPLETED))
		})

		It("returns polling ID when parsing fails", func(ctx SpecContext) {
			req := models.CreateConversionRequest{
				Conversion: models.PSPConversion{
					WalletID:     "wallet-123",
					SourceAsset:  "BTC/8",
					TargetAsset:  "USD/2",
					SourceAmount: big.NewInt(100000000),
				},
			}

			m.EXPECT().CreateMarketSellOrder(gomock.Any(), gomock.Any()).Return(
				&client.CreateOrderResponse{
					ID:       "123456789",
					DateTime: "2024-01-01 00:00:00",
					Type:     "1",
					Price:    "invalid-price",
					Amount:   "1.00000000",
				},
				nil,
			)

			res, err := plg.CreateConversion(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Conversion).To(BeNil())
			Expect(res.PollingConversionID).NotTo(BeNil())
			Expect(*res.PollingConversionID).To(Equal("123456789"))
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

			m.EXPECT().CreateMarketSellOrder(gomock.Any(), gomock.Any()).Return(
				nil,
				fmt.Errorf("client error"),
			)

			_, err := plg.CreateConversion(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to create conversion"))
		})

		It("handles invalid datetime gracefully", func(ctx SpecContext) {
			req := models.CreateConversionRequest{
				Conversion: models.PSPConversion{
					WalletID:     "wallet-123",
					SourceAsset:  "ETH/18",
					TargetAsset:  "EUR/2",
					SourceAmount: big.NewInt(1000000000000000000), // 1 ETH
				},
			}

			m.EXPECT().CreateMarketSellOrder(gomock.Any(), gomock.Any()).Return(
				&client.CreateOrderResponse{
					ID:       "987654321",
					DateTime: "invalid-datetime",
					Type:     "1",
					Price:    "2500.00",
					Amount:   "1.00000000",
				},
				nil,
			)

			res, err := plg.CreateConversion(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Conversion).NotTo(BeNil())
			Expect(res.Conversion.Reference).To(Equal("987654321"))
			// CreatedAt should default to current time when parsing fails
			Expect(res.Conversion.CreatedAt).NotTo(BeZero())
		})
	})

	Context("helper functions", func() {
		It("strips asset precision correctly", func() {
			Expect(stripAssetPrecision("BTC/8")).To(Equal("BTC"))
			Expect(stripAssetPrecision("USD/2")).To(Equal("USD"))
			Expect(stripAssetPrecision("EUR")).To(Equal("EUR"))
		})
	})
})
