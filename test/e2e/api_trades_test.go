//go:build it

package test_suite

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/publish"
	"github.com/formancehq/go-libs/v3/testing/deferred"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/client/models/components"
	evts "github.com/formancehq/payments/pkg/events"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/shopspring/decimal"

	. "github.com/formancehq/payments/pkg/testserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("Payments API Trades", Serial, func() {
	var (
		db  = UseTemplatedDatabase()
		ctx = logging.TestingContext()

		app *deferred.Deferred[*Server]
	)

	app = NewTestServer(func() Configuration {
		return Configuration{
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

	const (
		baseAsset    = "EUR/2"
		quoteAsset   = "USD/2"
		eventTimeout = 5 * time.Second
	)

	When("creating a trade with inline payments", func() {
		var (
			connectorID string
			e           chan *nats.Msg
			err         error
		)

		BeforeEach(func() {
			e = Subscribe(GinkgoT(), app.GetValue())
			connectorID, err = installConnector(ctx, app.GetValue(), uuid.New(), 3)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should create two payments with expected values", func() {
			createdAt := time.Now().UTC().Truncate(time.Second)
			portfolioAccountID, err := createV3Account(ctx, app.GetValue(), &components.V3CreateAccountRequest{
				Reference:    fmt.Sprintf("portfolio-%s", uuid.NewString()),
				ConnectorID:  connectorID,
				CreatedAt:    createdAt.Add(-time.Minute),
				AccountName:  "portfolio",
				Type:         "INTERNAL",
				DefaultAsset: pointer.For(quoteAsset),
				Metadata: map[string]string{
					"purpose": "trade-tests",
				},
			})
			Expect(err).To(BeNil())
			Eventually(e, eventTimeout).Should(Receive(Event(evts.EventTypeSavedAccounts)))

			tradeReference := fmt.Sprintf("trade-%s", uuid.NewString())
			price := "100.5"
			quantity := "1"
			quoteAmount := "100.5"
			feeAmount := "0.5"

			payload := map[string]any{
				"reference":          tradeReference,
				"connectorID":        connectorID,
				"createdAt":          createdAt.Format(time.RFC3339Nano),
				"portfolioAccountID": portfolioAccountID,
				"instrumentType":     "SPOT",
				"executionModel":     "ORDER_BOOK",
				"market": map[string]any{
					"symbol":     "EUR-USD",
					"baseAsset":  baseAsset,
					"quoteAsset": quoteAsset,
				},
				"side":   "BUY",
				"status": "FILLED",
				"requested": map[string]any{
					"quantity": quantity,
				},
				"executed": map[string]any{
					"quantity":     quantity,
					"quoteAmount":  quoteAmount,
					"averagePrice": price,
					"completedAt":  createdAt.Add(time.Second).Format(time.RFC3339Nano),
				},
				"fees": []map[string]any{
					{
						"asset":     quoteAsset,
						"amount":    feeAmount,
						"kind":      "TAKER",
						"appliedOn": "QUOTE",
					},
				},
				"fills": []map[string]any{
					{
						"tradeReference": "fill-1",
						"timestamp":      createdAt.Add(10 * time.Millisecond).Format(time.RFC3339Nano),
						"price":          price,
						"quantity":       quantity,
						"quoteAmount":    quoteAmount,
						"fees": []map[string]any{
							{
								"asset":     quoteAsset,
								"amount":    feeAmount,
								"kind":      "TAKER",
								"appliedOn": "QUOTE",
							},
						},
						"raw": map[string]string{
							"source": "fill",
						},
					},
				},
				"metadata": map[string]string{
					"test": "trade-e2e",
				},
				"raw": map[string]string{
					"origin": "test",
				},
				"createPayments": true,
			}

			// Buffer to hold events that don't match current search
			eventBuffer := make([]publish.EventMessage, 0)

			waitForEvent := func(eventType string, matcher func(publish.EventMessage) bool) {
				// First check buffered events
				for i, ev := range eventBuffer {
					if ev.Type == eventType {
						if matcher == nil || matcher(ev) {
							// Remove from buffer
							eventBuffer = append(eventBuffer[:i], eventBuffer[i+1:]...)
							return
						}
					}
				}

				// Then wait for new events
				timer := time.NewTimer(eventTimeout)
				defer timer.Stop()
				for {
					select {
					case msg := <-e:
						if !timer.Stop() {
							select {
							case <-timer.C:
							default:
							}
						}
						timer.Reset(eventTimeout)

						var ev publish.EventMessage
						Expect(json.Unmarshal(msg.Data, &ev)).To(Succeed())
						if ev.Type == eventType {
							if matcher == nil || matcher(ev) {
								return
							}
						} else {
							// Buffer this event for later
							eventBuffer = append(eventBuffer, ev)
						}
					case <-timer.C:
						Fail(fmt.Sprintf("timed out waiting for event %s", eventType))
					}
				}
			}

			decodePayload := func(ev publish.EventMessage, out any) {
				payloadBytes, err := json.Marshal(ev.Payload)
				Expect(err).To(BeNil())
				Expect(json.Unmarshal(payloadBytes, out)).To(Succeed())
			}

			type tradePayload struct {
				Reference string `json:"reference"`
			}
			type paymentPayload struct {
				Reference string            `json:"reference"`
				Metadata  map[string]string `json:"metadata"`
			}

			var createResp struct {
				Data struct {
					ID   string `json:"id"`
					Legs []struct {
						Role      string  `json:"role"`
						Asset     string  `json:"asset"`
						NetAmount string  `json:"netAmount"`
						Status    *string `json:"status"`
					} `json:"legs"`
				} `json:"data"`
			}
			err = app.GetValue().Client().Do(ctx, http.MethodPost, "/v3/trades", payload, &createResp)
			Expect(err).To(BeNil())
			Expect(createResp.Data.ID).NotTo(BeEmpty())

			tradeID := createResp.Data.ID
			baseRef := fmt.Sprintf("trade:%s:BASE", tradeID)
			quoteRef := fmt.Sprintf("trade:%s:QUOTE", tradeID)

			waitForEvent(evts.EventTypeSavedTrade, func(ev publish.EventMessage) bool {
				var payload tradePayload
				decodePayload(ev, &payload)
				return payload.Reference == tradeReference
			})

			waitForEvent(evts.EventTypeSavedPayments, func(ev publish.EventMessage) bool {
				var payload paymentPayload
				decodePayload(ev, &payload)
				return payload.Reference == baseRef && payload.Metadata["tradeID"] == tradeID
			})

			waitForEvent(evts.EventTypeSavedPayments, func(ev publish.EventMessage) bool {
				var payload paymentPayload
				decodePayload(ev, &payload)
				return payload.Reference == quoteRef && payload.Metadata["tradeID"] == tradeID
			})

			connector := models.MustConnectorIDFromString(connectorID)
			basePaymentID := models.PaymentID{
				PaymentReference: models.PaymentReference{
					Reference: baseRef,
					Type:      models.PAYMENT_TYPE_PAYIN,
				},
				ConnectorID: connector,
			}
			quotePaymentID := models.PaymentID{
				PaymentReference: models.PaymentReference{
					Reference: quoteRef,
					Type:      models.PAYMENT_TYPE_PAYOUT,
				},
				ConnectorID: connector,
			}

			Expect(createResp.Data.Legs).To(HaveLen(2))
			legsByRole := map[string]struct {
				Role      string
				Asset     string
				NetAmount string
				Status    *string
			}{}
			for _, leg := range createResp.Data.Legs {
				legsByRole[leg.Role] = struct {
					Role      string
					Asset     string
					NetAmount string
					Status    *string
				}{
					Role:      leg.Role,
					Asset:     leg.Asset,
					NetAmount: leg.NetAmount,
					Status:    leg.Status,
				}
			}

			baseLeg, ok := legsByRole["BASE"]
			Expect(ok).To(BeTrue())
			Expect(baseLeg.Asset).To(Equal(baseAsset))
			Expect(baseLeg.Status).NotTo(BeNil())
			Expect(*baseLeg.Status).To(Equal("SUCCEEDED"))
			baseNet := decimal.RequireFromString(baseLeg.NetAmount)
			Expect(baseNet).To(Equal(decimal.RequireFromString("1")))

			quoteLeg, ok := legsByRole["QUOTE"]
			Expect(ok).To(BeTrue())
			Expect(quoteLeg.Asset).To(Equal(quoteAsset))
			Expect(quoteLeg.Status).NotTo(BeNil())
			Expect(*quoteLeg.Status).To(Equal("SUCCEEDED"))
			quoteNet := decimal.RequireFromString(quoteLeg.NetAmount)
			Expect(quoteNet).To(Equal(decimal.RequireFromString("101")))

			basePaymentResp, err := app.GetValue().SDK().Payments.V3.GetPayment(ctx, basePaymentID.String())
			Expect(err).To(BeNil())
			basePayment := basePaymentResp.GetV3GetPaymentResponse().Data
			Expect(basePayment.Asset).To(Equal(baseAsset))
			Expect(basePayment.Type).To(Equal(components.V3PaymentTypeEnumPayIn))
			Expect(basePayment.Scheme).To(Equal(models.PAYMENT_SCHEME_EXCHANGE.String()))
			Expect(basePayment.DestinationAccountID).NotTo(BeNil())
			Expect(*basePayment.DestinationAccountID).To(Equal(portfolioAccountID))
			Expect(decimal.NewFromBigInt(basePayment.Amount, 0)).To(Equal(baseNet))

			quotePaymentResp, err := app.GetValue().SDK().Payments.V3.GetPayment(ctx, quotePaymentID.String())
			Expect(err).To(BeNil())
			quotePayment := quotePaymentResp.GetV3GetPaymentResponse().Data
			Expect(quotePayment.Asset).To(Equal(quoteAsset))
			Expect(quotePayment.Type).To(Equal(components.V3PaymentTypeEnumPayout))
			Expect(quotePayment.Scheme).To(Equal(models.PAYMENT_SCHEME_EXCHANGE.String()))
			Expect(quotePayment.SourceAccountID).NotTo(BeNil())
			Expect(*quotePayment.SourceAccountID).To(Equal(portfolioAccountID))
			Expect(decimal.NewFromBigInt(quotePayment.Amount, 0)).To(Equal(quoteNet))
		})
	})
})
