//go:build it

package test_suite

import (
	"encoding/json"
	"path/filepath"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/publish"
	"github.com/formancehq/go-libs/v3/testing/deferred"
	"github.com/formancehq/payments/internal/models"
	evts "github.com/formancehq/payments/pkg/events"
	"github.com/formancehq/payments/pkg/testserver"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("DummyPay Trades Ingestion", Serial, func() {
	var (
		db  = UseTemplatedDatabase()
		ctx = logging.TestingContext()

		app *deferred.Deferred[*testserver.Server]
	)

	app = testserver.NewTestServer(func() testserver.Configuration {
		return testserver.Configuration{
			Stack:                 stack,
			PostgresConfiguration: db.GetValue().ConnectionOptions(),
			NatsURL:               natsServer.GetValue().ClientURL(),
			TemporalNamespace:     temporalServer.GetValue().DefaultNamespace(),
			TemporalAddress:       temporalServer.GetValue().Address(),
			Output:                GinkgoWriter,
		}
	})

	AfterEach(func() {
		flushRemainingWorkflows(ctx)
	})

	It("should ingest trades from trades.json", func() {
		e := testserver.Subscribe(GinkgoT(), app.GetValue())

		// Create a temporary directory for the connector
		dir := GinkgoT().TempDir()

		// Create trades.json BEFORE installing the connector so it's available immediately
		tradeRef := uuid.NewString()
		trades := []models.PSPTrade{
			{
				Reference:      tradeRef,
				CreatedAt:      time.Now().UTC(),
				InstrumentType: models.TRADE_INSTRUMENT_TYPE_SPOT,
				ExecutionModel: models.TRADE_EXECUTION_MODEL_ORDER_BOOK,
				Market: models.TradeMarket{
					Symbol:     "EUR-USD",
					BaseAsset:  "EUR/2",
					QuoteAsset: "USD/2",
				},
				Side:   models.TRADE_SIDE_BUY,
				Status: models.TRADE_STATUS_FILLED,
				Executed: models.TradeExecuted{
					Quantity:     func() *string { s := "100"; return &s }(),
					QuoteAmount:  func() *string { s := "110"; return &s }(),
					AveragePrice: func() *string { s := "1.1"; return &s }(),
					CompletedAt:  func() *time.Time { t := time.Now().UTC(); return &t }(),
				},
				Fills: []models.TradeFill{
					{
						TradeReference: tradeRef,
						Timestamp:      time.Now().UTC(),
						Price:          "1.1",
						Quantity:       "100",
						QuoteAmount:    "110",
						Fees:           []models.TradeFee{},
						Raw:            json.RawMessage(`{}`),
					},
				},
				Fees:     []models.TradeFee{},
				Raw:      json.RawMessage(`{}`),
				Metadata: map[string]string{},
			},
		}

		tradesBytes, err := json.Marshal(trades)
		Expect(err).To(BeNil())

		err = testserver.WriteFile(filepath.Join(dir, "trades.json"), tradesBytes)
		Expect(err).To(BeNil())

		// Now install the connector with the trades.json file already present
		config := map[string]any{
			"directory":     dir,
			"name":          "dummypay-test",
			"pollingPeriod": "20m", // Minimum allowed polling period
		}

		var installResp struct {
			Data string `json:"data"`
		}
		err = app.GetValue().Client().Do(ctx, "POST", "/v3/connectors/install/dummypay", config, &installResp)
		Expect(err).To(BeNil())
		connectorID := installResp.Data

		// Wait for the trade to be ingested - may take a while for schedule to trigger
		Eventually(func() bool {
			select {
			case msg := <-e:
				var ev publish.EventMessage
				if err := json.Unmarshal(msg.Data, &ev); err != nil {
					return false
				}
				if ev.Type == evts.EventTypeSavedTrade {
					payloadBytes, err := json.Marshal(ev.Payload)
					if err != nil {
						return false
					}
					var payload struct {
						Reference   string `json:"reference"`
						ConnectorID string `json:"connectorID"`
					}
					if err := json.Unmarshal(payloadBytes, &payload); err != nil {
						return false
					}
					return payload.Reference == tradeRef && payload.ConnectorID == connectorID
				}
			default:
			}
			return false
		}, 30*time.Second, 500*time.Millisecond).Should(BeTrue())
	})
})
