//go:build it

package test_suite

import (
	"math/big"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/testing/deferred"
	"github.com/google/uuid"

	. "github.com/formancehq/payments/pkg/testserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// CreateConversionRequest matches the API request structure
type CreateConversionRequest struct {
	Reference    string            `json:"reference"`
	ConnectorID  string            `json:"connectorID"`
	SourceAsset  string            `json:"sourceAsset"`
	TargetAsset  string            `json:"targetAsset"`
	SourceAmount *big.Int          `json:"sourceAmount"`
	WalletID     string            `json:"walletId"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// ConversionResponse matches the create/get conversion API response structure
type ConversionResponse struct {
	Data struct {
		ID           string            `json:"id"`
		Reference    string            `json:"reference"`
		ConnectorID  string            `json:"connectorID"`
		SourceAsset  string            `json:"sourceAsset"`
		TargetAsset  string            `json:"targetAsset"`
		SourceAmount *big.Int          `json:"sourceAmount"`
		TargetAmount *big.Int          `json:"targetAmount,omitempty"`
		Status       string            `json:"status"`
		WalletID     string            `json:"walletId"`
		Metadata     map[string]string `json:"metadata,omitempty"`
		CreatedAt    time.Time         `json:"createdAt"`
		UpdatedAt    time.Time         `json:"updatedAt"`
	} `json:"data"`
}

// CreateConversionDirectResponse matches the direct conversion API response (without data wrapper)
type CreateConversionDirectResponse struct {
	ID           string            `json:"id"`
	Reference    string            `json:"reference"`
	ConnectorID  string            `json:"connectorID"`
	SourceAsset  string            `json:"sourceAsset"`
	TargetAsset  string            `json:"targetAsset"`
	SourceAmount *big.Int          `json:"sourceAmount"`
	TargetAmount *big.Int          `json:"targetAmount,omitempty"`
	Status       string            `json:"status"`
	WalletID     string            `json:"walletId"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
}

// ConversionsListResponse matches the list conversions API response structure
type ConversionsListResponse struct {
	Cursor struct {
		PageSize int  `json:"pageSize"`
		HasMore  bool `json:"hasMore"`
		Data     []struct {
			ID           string            `json:"id"`
			Reference    string            `json:"reference"`
			ConnectorID  string            `json:"connectorID"`
			SourceAsset  string            `json:"sourceAsset"`
			TargetAsset  string            `json:"targetAsset"`
			SourceAmount *big.Int          `json:"sourceAmount"`
			TargetAmount *big.Int          `json:"targetAmount,omitempty"`
			Status       string            `json:"status"`
			WalletID     string            `json:"walletId"`
			Metadata     map[string]string `json:"metadata,omitempty"`
			CreatedAt    time.Time         `json:"createdAt"`
			UpdatedAt    time.Time         `json:"updatedAt"`
		} `json:"data"`
	} `json:"cursor"`
}

var _ = Context("Payments API Conversions", Serial, func() {
	var (
		db  = UseTemplatedDatabase()
		ctx = logging.TestingContext()

		app *deferred.Deferred[*Server]
	)

	app = NewTestServer(func() Configuration {
		return Configuration{
			Stack:                      stack,
			PostgresConfiguration:      db.GetValue().ConnectionOptions(),
			NatsURL:                    natsServer.GetValue().ClientURL(),
			TemporalNamespace:          temporalServer.GetValue().DefaultNamespace(),
			TemporalAddress:            temporalServer.GetValue().Address(),
			Output:                     GinkgoWriter,
			SkipOutboxScheduleCreation: true,
		}
	})

	AfterEach(func() {
		flushRemainingWorkflows(ctx)
	})

	When("creating and managing conversions", func() {
		var (
			connectorID string
			err         error
		)

		BeforeEach(func() {
			connectorID, err = installConnector(ctx, app.GetValue(), uuid.New(), 3)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should create a conversion successfully", func() {
			createRequest := CreateConversionRequest{
				Reference:    "test-conversion-" + uuid.New().String()[:8],
				ConnectorID:  connectorID,
				SourceAsset:  "USD/2",
				TargetAsset:  "BTC/8",
				SourceAmount: big.NewInt(100000),
				WalletID:     "test-wallet-" + uuid.New().String()[:8],
				Metadata:     map[string]string{"test": "value"},
			}

			var createResponse CreateConversionDirectResponse
			err := app.GetValue().Client().Do(ctx, http.MethodPost, "/v3/conversions", createRequest, &createResponse)
			Expect(err).To(BeNil())
			Expect(createResponse.Reference).To(Equal(createRequest.Reference))
			Expect(createResponse.Status).To(Equal("PENDING"))
			Expect(createResponse.SourceAsset).To(Equal("USD/2"))
			Expect(createResponse.TargetAsset).To(Equal("BTC/8"))
			Expect(createResponse.WalletID).To(Equal(createRequest.WalletID))
		})

		It("should get a conversion by ID", func() {
			// First create a conversion
			createRequest := CreateConversionRequest{
				Reference:    "test-get-conversion-" + uuid.New().String()[:8],
				ConnectorID:  connectorID,
				SourceAsset:  "EUR/2",
				TargetAsset:  "ETH/8",
				SourceAmount: big.NewInt(500000),
				WalletID:     "test-wallet-" + uuid.New().String()[:8],
			}

			var createResponse CreateConversionDirectResponse
			err := app.GetValue().Client().Do(ctx, http.MethodPost, "/v3/conversions", createRequest, &createResponse)
			Expect(err).To(BeNil())

			// Now get the conversion
			conversionID := createResponse.ID
			var getResponse ConversionResponse
			err = app.GetValue().Client().Get(ctx, "/v3/conversions/"+conversionID, &getResponse)
			Expect(err).To(BeNil())
			Expect(getResponse.Data.ID).To(Equal(conversionID))
			Expect(getResponse.Data.Reference).To(Equal(createRequest.Reference))
			Expect(getResponse.Data.ConnectorID).To(Equal(connectorID))
		})

		It("should list conversions with pagination", func() {
			// Create multiple conversions
			for i := 0; i < 3; i++ {
				createRequest := CreateConversionRequest{
					Reference:    "test-list-conversion-" + uuid.New().String()[:8],
					ConnectorID:  connectorID,
					SourceAsset:  "USD/2",
					TargetAsset:  "BTC/8",
					SourceAmount: big.NewInt(int64(100000 * (i + 1))),
					WalletID:     "test-wallet-" + uuid.New().String()[:8],
				}

				var createResponse CreateConversionDirectResponse
				err := app.GetValue().Client().Do(ctx, http.MethodPost, "/v3/conversions", createRequest, &createResponse)
				Expect(err).To(BeNil())
			}

			// List conversions
			var listResponse ConversionsListResponse
			err := app.GetValue().Client().Get(ctx, "/v3/conversions", &listResponse)
			Expect(err).To(BeNil())
			Expect(len(listResponse.Cursor.Data)).To(BeNumerically(">=", 3))
		})

		It("should reject conversion without required fields", func() {
			// Missing walletId
			createRequest := CreateConversionRequest{
				Reference:    "test-invalid-conversion-" + uuid.New().String()[:8],
				ConnectorID:  connectorID,
				SourceAsset:  "USD/2",
				TargetAsset:  "BTC/8",
				SourceAmount: big.NewInt(100000),
				// Missing WalletID
			}

			var createResponse CreateConversionDirectResponse
			err := app.GetValue().Client().Do(ctx, http.MethodPost, "/v3/conversions", createRequest, &createResponse)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("400"))
		})

		It("should reject conversion with invalid connector ID", func() {
			createRequest := CreateConversionRequest{
				Reference:    "test-invalid-connector-" + uuid.New().String()[:8],
				ConnectorID:  "invalid-connector-id",
				SourceAsset:  "USD/2",
				TargetAsset:  "BTC/8",
				SourceAmount: big.NewInt(100000),
				WalletID:     "test-wallet-" + uuid.New().String()[:8],
			}

			var createResponse CreateConversionDirectResponse
			err := app.GetValue().Client().Do(ctx, http.MethodPost, "/v3/conversions", createRequest, &createResponse)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("400"))
		})

		It("should create multiple conversions with different asset pairs", func() {
			assetPairs := []struct {
				source string
				target string
			}{
				{"USD/2", "BTC/8"},
				{"EUR/2", "ETH/8"},
				{"GBP/2", "USDT/6"},
			}

			for _, pair := range assetPairs {
				createRequest := CreateConversionRequest{
					Reference:    "test-pair-conversion-" + uuid.New().String()[:8],
					ConnectorID:  connectorID,
					SourceAsset:  pair.source,
					TargetAsset:  pair.target,
					SourceAmount: big.NewInt(100000),
					WalletID:     "test-wallet-" + uuid.New().String()[:8],
				}

				var createResponse CreateConversionDirectResponse
				err := app.GetValue().Client().Do(ctx, http.MethodPost, "/v3/conversions", createRequest, &createResponse)
				Expect(err).To(BeNil())
				Expect(createResponse.SourceAsset).To(Equal(pair.source))
				Expect(createResponse.TargetAsset).To(Equal(pair.target))
			}
		})
	})
})
