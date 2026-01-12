package coinbaseprime

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/public/coinbaseprime/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Coinbase Prime Plugin Conversions", func() {
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
			config: Config{
				PortfolioID: "test-portfolio-id",
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetch next conversions", func() {
		It("returns empty list as not implemented via dedicated endpoint", func(ctx SpecContext) {
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
					SourceAsset:  "USDC",
					TargetAsset:  "USD",
					SourceAmount: big.NewInt(100000000), // 100 USDC
				},
			}

			m.EXPECT().CreateConversion(gomock.Any(), gomock.Any()).Return(
				&client.CreateConversionResponse{
					Conversion: client.Conversion{
						ID:           "conversion-id-1",
						PortfolioID:  "test-portfolio-id",
						WalletID:     "wallet-123",
						SourceSymbol: "USDC",
						TargetSymbol: "USD",
						SourceAmount: "100.00",
						TargetAmount: "100.00",
						Status:       "COMPLETED",
						CreatedAt:    time.Now(),
					},
				},
				nil,
			)

			res, err := plg.CreateConversion(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Conversion).NotTo(BeNil())
			Expect(res.Conversion.Reference).To(Equal("conversion-id-1"))
			Expect(res.Conversion.Status).To(Equal(models.CONVERSION_STATUS_COMPLETED))
		})

		It("returns polling ID when conversion creation succeeds but parsing fails", func(ctx SpecContext) {
			req := models.CreateConversionRequest{
				Conversion: models.PSPConversion{
					WalletID:     "wallet-123",
					SourceAsset:  "USDC",
					TargetAsset:  "USD",
					SourceAmount: big.NewInt(100000000),
				},
			}

			m.EXPECT().CreateConversion(gomock.Any(), gomock.Any()).Return(
				&client.CreateConversionResponse{
					Conversion: client.Conversion{
						ID:           "conversion-id-1",
						PortfolioID:  "test-portfolio-id",
						WalletID:     "wallet-123",
						SourceSymbol: "USDC",
						TargetSymbol: "USD",
						SourceAmount: "invalid-amount",
						TargetAmount: "100.00",
						Status:       "PENDING",
						CreatedAt:    time.Now(),
					},
				},
				nil,
			)

			res, err := plg.CreateConversion(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Conversion).To(BeNil())
			Expect(res.PollingConversionID).NotTo(BeNil())
			Expect(*res.PollingConversionID).To(Equal("conversion-id-1"))
		})

		It("returns error on client failure", func(ctx SpecContext) {
			req := models.CreateConversionRequest{
				Conversion: models.PSPConversion{
					WalletID:     "wallet-123",
					SourceAsset:  "USDC",
					TargetAsset:  "USD",
					SourceAmount: big.NewInt(100000000),
				},
			}

			m.EXPECT().CreateConversion(gomock.Any(), gomock.Any()).Return(
				nil,
				fmt.Errorf("client error"),
			)

			_, err := plg.CreateConversion(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to create conversion"))
		})
	})

	Context("status mapping", func() {
		It("maps PENDING status correctly", func() {
			status := mapConversionStatus("PENDING")
			Expect(status).To(Equal(models.CONVERSION_STATUS_PENDING))
		})

		It("maps COMPLETED status correctly", func() {
			status := mapConversionStatus("COMPLETED")
			Expect(status).To(Equal(models.CONVERSION_STATUS_COMPLETED))
		})

		It("maps FAILED status correctly", func() {
			status := mapConversionStatus("FAILED")
			Expect(status).To(Equal(models.CONVERSION_STATUS_FAILED))
		})

		It("maps unknown status to PENDING", func() {
			status := mapConversionStatus("UNKNOWN")
			Expect(status).To(Equal(models.CONVERSION_STATUS_PENDING))
		})

		It("handles lowercase status", func() {
			status := mapConversionStatus("completed")
			Expect(status).To(Equal(models.CONVERSION_STATUS_COMPLETED))
		})
	})
})
