package workflow

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
	"go.temporal.io/sdk/workflow"
)

type FetchNextTrades struct {
	Config       models.Config      `json:"config"`
	ConnectorID  models.ConnectorID `json:"connectorID"`
	FromPayload  *FromPayload       `json:"fromPayload"`
	Periodically bool               `json:"periodically"`
}

func (w Workflow) runFetchNextTrades(
	ctx workflow.Context,
	fetchNextTrades FetchNextTrades,
	nextTasks []models.ConnectorTaskTree,
) error {
	if err := w.createInstance(ctx, fetchNextTrades.ConnectorID); err != nil {
		return errors.Wrap(err, "creating instance")
	}
	err := w.fetchNextTrades(ctx, fetchNextTrades, nextTasks)
	return w.terminateInstance(ctx, fetchNextTrades.ConnectorID, err)
}

func (w Workflow) fetchNextTrades(
	ctx workflow.Context,
	fetchNextTrades FetchNextTrades,
	nextTasks []models.ConnectorTaskTree,
) error {
	stateReference := models.CAPABILITY_FETCH_TRADES.String()
	if fetchNextTrades.FromPayload != nil {
		stateReference = fmt.Sprintf("%s-%s", models.CAPABILITY_FETCH_TRADES.String(), fetchNextTrades.FromPayload.ID)
	}

	stateID := models.StateID{
		Reference:   stateReference,
		ConnectorID: fetchNextTrades.ConnectorID,
	}
	state, err := activities.StorageStatesGet(infiniteRetryContext(ctx), stateID)
	if err != nil {
		return fmt.Errorf("retrieving state %s: %v", stateID.String(), err)
	}

	hasMore := true
	for hasMore {
		tradesResponse, err := activities.PluginFetchNextTrades(
			fetchNextActivityRetryContext(ctx),
			fetchNextTrades.ConnectorID,
			fetchNextTrades.FromPayload.GetPayload(),
			state.State,
			fetchNextTrades.Config.PageSize,
			fetchNextTrades.Periodically,
		)
		if err != nil {
			return errors.Wrap(err, "fetching next trades")
		}

		trades, err := models.FromPSPTrades(tradesResponse.Trades, fetchNextTrades.ConnectorID)
		if err != nil {
			return errors.Wrap(err, "mapping trades")
		}

		if len(trades) > 0 {
			// STEP 1: Store trades first
			// This must happen before payment generation to ensure trade IDs exist in the database
			// for foreign key constraints and referential integrity.
			err = activities.StorageTradesStore(
				infiniteRetryContext(ctx),
				trades,
			)
			if err != nil {
				return errors.Wrap(err, "storing next trades")
			}
		}

		// STEP 2: Generate and store payments from trades
		// For PSPs like Bitstamp, a "transaction" can be a trade that needs to generate
		// two payments (one for base asset, one for quote asset). This ensures atomicity:
		// the trade and its associated payments are both persisted before events are sent.
		//
		// Why generate payments here instead of in the plugin?
		// 1. Separation of concerns: plugins fetch raw PSP data, workflows handle business logic
		// 2. Consistency: payment generation rules are centralized, not duplicated per connector
		// 3. Database atomicity: Temporal's infinite retry ensures both trades and payments
		//    are committed before proceeding to events
		var paymentsToCreate []models.Payment
		var tradesWithLegs []models.Trade // Trades that need leg updates after payments are stored

		for _, trade := range trades {
			// Only generate payments if the trade has a portfolio account
			// This is required because payments need source/destination accounts
			if trade.PortfolioAccountID != nil {
				basePayment, quotePayment, err := models.CreatePaymentsFromTrade(trade, *trade.PortfolioAccountID)
				if err != nil {
					// Log but don't fail - some trades may not be eligible for payment generation
					// (e.g., pending/cancelled trades, trades with missing execution data)
					workflow.GetLogger(ctx).Warn("Failed to create payments from trade, skipping payment generation",
						"trade_id", trade.ID.String(),
						"error", err.Error())
					continue
				}

				// Collect payments to be stored in batch
				paymentsToCreate = append(paymentsToCreate, basePayment, quotePayment)
				tradesWithLegs = append(tradesWithLegs, trade)
			}
		}

		// Store all generated payments atomically
		// Using infiniteRetryContext ensures this will retry until successful,
		// maintaining consistency between trades and their payments
		if len(paymentsToCreate) > 0 {
			err = activities.StoragePaymentsStore(
				infiniteRetryContext(ctx),
				paymentsToCreate,
			)
			if err != nil {
				return errors.Wrap(err, "storing payments generated from trades")
			}

			// STEP 3: Update trades with payment leg references
			// Now that payments are persisted with their IDs, we can link them back to the trade
			// This creates the bidirectional relationship: Trade -> Payment (via legs) and
			// Payment -> Trade (via metadata)
			for i, trade := range tradesWithLegs {
				basePaymentIdx := i * 2      // Each trade generates 2 payments
				quotePaymentIdx := i*2 + 1
				
				basePayment := paymentsToCreate[basePaymentIdx]
				quotePayment := paymentsToCreate[quotePaymentIdx]

				// Build trade legs that reference the payment IDs
				// Direction semantics: CREDIT = funds coming in, DEBIT = funds going out
				trade.Legs = []models.TradeLeg{
					{
						Role:      models.TRADE_LEG_ROLE_BASE,
						Direction: func() models.TradeLegDirection {
							if trade.Side == models.TRADE_SIDE_BUY {
								return models.TRADE_LEG_DIRECTION_CREDIT // Buying = receiving base asset
							}
							return models.TRADE_LEG_DIRECTION_DEBIT // Selling = giving base asset
						}(),
						Asset:     trade.Market.BaseAsset,
						NetAmount: basePayment.Adjustments[0].Amount.String(), // Use first adjustment amount
						PaymentID: &basePayment.ID,
						Status:    &basePayment.Status,
					},
					{
						Role:      models.TRADE_LEG_ROLE_QUOTE,
						Direction: func() models.TradeLegDirection {
							if trade.Side == models.TRADE_SIDE_BUY {
								return models.TRADE_LEG_DIRECTION_DEBIT // Buying = spending quote asset
							}
							return models.TRADE_LEG_DIRECTION_CREDIT // Selling = receiving quote asset
						}(),
						Asset:     trade.Market.QuoteAsset,
						NetAmount: quotePayment.Adjustments[0].Amount.String(),
						PaymentID: &quotePayment.ID,
						Status:    &quotePayment.Status,
					},
				}

				// Update the trade in the original slice so events are sent with complete data
				for j := range trades {
					if trades[j].ID == trade.ID {
						trades[j].Legs = trade.Legs
						break
					}
				}
			}

			// Store trades again with updated leg information
			// This is an upsert operation, so it's safe to call multiple times (idempotent)
			if len(tradesWithLegs) > 0 {
				err = activities.StorageTradesStore(
					infiniteRetryContext(ctx),
					trades, // Use full trades slice with updated legs
				)
				if err != nil {
					return errors.Wrap(err, "updating trades with payment legs")
				}
			}
		}

		// STEP 4: Send events for both trades and payments
		// Events are sent AFTER all database operations succeed, ensuring subscribers
		// receive notifications only for fully persisted, consistent data.
		//
		// We send events in parallel for performance, but the WaitGroup ensures
		// all events are sent before proceeding to the next batch of trades.
		wg := workflow.NewWaitGroup(ctx)
		// Calculate channel size: 1 trade event + potentially 2 payment events per trade
		maxEvents := len(trades) * 3
		errChan := make(chan error, maxEvents)

		// Send trade events
		for _, trade := range trades {
			t := trade
			wg.Add(1)
			workflow.Go(ctx, func(ctx workflow.Context) {
				defer wg.Done()

				if err := w.runSendEvents(ctx, SendEvents{
					Trade: &t,
				}); err != nil {
					errChan <- errors.Wrap(err, "sending trade event")
				}
			})
		}

		// Send payment events for generated payments
		// Each payment has adjustments, and we send one event per adjustment (v3 API contract)
		for _, payment := range paymentsToCreate {
			p := payment
			wg.Add(1)
			workflow.Go(ctx, func(ctx workflow.Context) {
				defer wg.Done()

				if err := w.runSendEvents(ctx, SendEvents{
					Payment: &p,
				}); err != nil {
					errChan <- errors.Wrap(err, "sending payment event")
				}
			})
		}

		wg.Wait(ctx)
		close(errChan)

		for err := range errChan {
			if err != nil {
				return err
			}
		}

		if len(nextTasks) > 0 {
			// Logic for next tasks if needed
		}

		if tradesResponse.HasMore {
			state.State = tradesResponse.NewState
			err = activities.StorageStatesStore(infiniteRetryContext(ctx), *state)
			if err != nil {
				return errors.Wrap(err, "updating state")
			}
		}

		hasMore = tradesResponse.HasMore
	}

	return nil
}

const RunFetchNextTrades = "FetchTrades"
