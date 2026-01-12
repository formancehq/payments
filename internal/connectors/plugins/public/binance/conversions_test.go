package binance

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/public/binance/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Binance Plugin Conversions", func() {
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
		It("returns empty list as Binance has no dedicated conversions endpoint", func(ctx SpecContext) {
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
					TargetAsset:  "USDT/6",
					SourceAmount: big.NewInt(100000000), // 1 BTC
				},
			}

			m.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).Return(
				&client.CreateOrderResponse{
					Symbol:             "BTCUSDT",
					OrderID:            12345,
					ClientOrderID:      "test-order",
					TransactTime:       1704067200000, // 2024-01-01 00:00:00 UTC
					Price:              "0.00000000",
					OrigQty:            "1.00000000",
					ExecutedQty:        "1.00000000",
					CumulativeQuoteQty: "42000.00000000",
					Status:             "FILLED",
					Type:               "MARKET",
					Side:               "SELL",
				},
				nil,
			)

			res, err := plg.CreateConversion(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Conversion).NotTo(BeNil())
			Expect(res.Conversion.Reference).To(Equal("12345"))
			Expect(res.Conversion.Status).To(Equal(models.CONVERSION_STATUS_COMPLETED))
		})

		It("returns polling ID when parsing fails", func(ctx SpecContext) {
			req := models.CreateConversionRequest{
				Conversion: models.PSPConversion{
					WalletID:     "wallet-123",
					SourceAsset:  "BTC/8",
					TargetAsset:  "USDT/6",
					SourceAmount: big.NewInt(100000000),
				},
			}

			m.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).Return(
				&client.CreateOrderResponse{
					Symbol:             "BTCUSDT",
					OrderID:            12345,
					ClientOrderID:      "test-order",
					TransactTime:       1704067200000,
					CumulativeQuoteQty: "invalid-amount",
					Status:             "FILLED",
				},
				nil,
			)

			res, err := plg.CreateConversion(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Conversion).To(BeNil())
			Expect(res.PollingConversionID).NotTo(BeNil())
			Expect(*res.PollingConversionID).To(Equal("12345"))
		})

		It("returns error on client failure", func(ctx SpecContext) {
			req := models.CreateConversionRequest{
				Conversion: models.PSPConversion{
					WalletID:     "wallet-123",
					SourceAsset:  "BTC/8",
					TargetAsset:  "USDT/6",
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

		It("handles PARTIALLY_FILLED status as pending", func(ctx SpecContext) {
			req := models.CreateConversionRequest{
				Conversion: models.PSPConversion{
					WalletID:     "wallet-123",
					SourceAsset:  "BTC/8",
					TargetAsset:  "USDT/6",
					SourceAmount: big.NewInt(100000000),
				},
			}

			m.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).Return(
				&client.CreateOrderResponse{
					Symbol:             "BTCUSDT",
					OrderID:            12345,
					ClientOrderID:      "test-order",
					TransactTime:       1704067200000,
					CumulativeQuoteQty: "21000.00000000",
					Status:             "PARTIALLY_FILLED",
				},
				nil,
			)

			res, err := plg.CreateConversion(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Conversion).NotTo(BeNil())
			Expect(res.Conversion.Status).To(Equal(models.CONVERSION_STATUS_PENDING))
		})

		It("handles REJECTED status as failed", func(ctx SpecContext) {
			req := models.CreateConversionRequest{
				Conversion: models.PSPConversion{
					WalletID:     "wallet-123",
					SourceAsset:  "BTC/8",
					TargetAsset:  "USDT/6",
					SourceAmount: big.NewInt(100000000),
				},
			}

			m.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).Return(
				&client.CreateOrderResponse{
					Symbol:             "BTCUSDT",
					OrderID:            12345,
					ClientOrderID:      "test-order",
					TransactTime:       1704067200000,
					CumulativeQuoteQty: "0.00000000",
					Status:             "REJECTED",
				},
				nil,
			)

			res, err := plg.CreateConversion(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Conversion).NotTo(BeNil())
			Expect(res.Conversion.Status).To(Equal(models.CONVERSION_STATUS_FAILED))
		})

		It("handles EXPIRED status as failed", func(ctx SpecContext) {
			req := models.CreateConversionRequest{
				Conversion: models.PSPConversion{
					WalletID:     "wallet-123",
					SourceAsset:  "BTC/8",
					TargetAsset:  "USDT/6",
					SourceAmount: big.NewInt(100000000),
				},
			}

			m.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).Return(
				&client.CreateOrderResponse{
					Symbol:             "BTCUSDT",
					OrderID:            12345,
					ClientOrderID:      "test-order",
					TransactTime:       1704067200000,
					CumulativeQuoteQty: "0.00000000",
					Status:             "EXPIRED",
				},
				nil,
			)

			res, err := plg.CreateConversion(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Conversion).NotTo(BeNil())
			Expect(res.Conversion.Status).To(Equal(models.CONVERSION_STATUS_FAILED))
		})
	})

	Context("helper functions", func() {
		It("strips asset precision correctly", func() {
			Expect(stripAssetPrecision("BTC/8")).To(Equal("BTC"))
			Expect(stripAssetPrecision("USDT/6")).To(Equal("USDT"))
			Expect(stripAssetPrecision("ETH")).To(Equal("ETH"))
		})
	})
})
