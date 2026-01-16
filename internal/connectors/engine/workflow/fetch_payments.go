package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type FetchNextPayments struct {
	ConnectorID  models.ConnectorID `json:"connectorID"`
	FromPayload  *FromPayload       `json:"fromPayload"`
	Periodically bool               `json:"periodically"`
}

func (w Workflow) runFetchNextPayments(
	ctx workflow.Context,
	fetchNextPayments FetchNextPayments,
	nextTasks []models.ConnectorTaskTree,
) error {
	if err := w.createInstance(ctx, fetchNextPayments.ConnectorID); err != nil {
		return errors.Wrap(err, "creating instance")
	}
	err := w.fetchNextPayments(ctx, fetchNextPayments, nextTasks)
	return w.terminateInstance(ctx, fetchNextPayments.ConnectorID, err)
}

func (w Workflow) fetchNextPayments(
	ctx workflow.Context,
	fetchNextPayments FetchNextPayments,
	nextTasks []models.ConnectorTaskTree,
) error {
	stateReference := models.CAPABILITY_FETCH_PAYMENTS.String()
	if fetchNextPayments.FromPayload != nil {
		stateReference = fmt.Sprintf("%s-%s", models.CAPABILITY_FETCH_PAYMENTS.String(), fetchNextPayments.FromPayload.ID)
	}

	stateID := models.StateID{
		Reference:   stateReference,
		ConnectorID: fetchNextPayments.ConnectorID,
	}
	state, err := activities.StorageStatesGet(infiniteRetryContext(ctx), stateID)
	if err != nil {
		return fmt.Errorf("retrieving state %s: %v", stateID.String(), err)
	}

	// Get pageSize from registry using provider from ConnectorID (no DB call needed)
	pageSize, err := registry.GetPageSize(fetchNextPayments.ConnectorID.Provider)
	if err != nil {
		return fmt.Errorf("getting page size: %w", err)
	}

	hasMore := true
	for hasMore {
		paymentsResponse, err := activities.PluginFetchNextPayments(
			infiniteRetryWithLongTimeoutContext(ctx),
			fetchNextPayments.ConnectorID,
			fetchNextPayments.FromPayload.GetPayload(),
			state.State,
			int(pageSize),
			fetchNextPayments.Periodically,
		)
		if err != nil {
			return errors.Wrap(err, "fetching next payments")
		}

		payments, err := models.FromPSPPayments(
			paymentsResponse.Payments,
			fetchNextPayments.ConnectorID,
		)
		if err != nil {
			return temporal.NewNonRetryableApplicationError(
				"failed to translate psp payments",
				ErrValidation,
				err,
			)
		}

		if len(payments) > 0 {
			err = activities.StoragePaymentsStore(
				infiniteRetryContext(ctx),
				payments,
			)
			if err != nil {
				return errors.Wrap(err, "storing next payments")
			}
		}

		wg := workflow.NewWaitGroup(ctx)
		errChan := make(chan error, len(paymentsResponse.Payments)*3)
		for _, payment := range payments {
			p := payment

			wg.Add(1)
			workflow.Go(ctx, func(ctx workflow.Context) {
				defer wg.Done()

				if IsPaymentInitiationUpdateOptimizationsEnabled(ctx) {
					if err := activities.StoragePaymentInitiationUpdateFromPayment(
						infiniteRetryContext(ctx),
						p.Status,
						p.CreatedAt,
						p.ID,
					); err != nil {
						errChan <- errors.Wrap(err, "updating payment initiation from payment")
					}
				} else {
					if err := workflow.ExecuteChildWorkflow(
						workflow.WithChildOptions(
							ctx,
							workflow.ChildWorkflowOptions{
								TaskQueue:         w.getDefaultTaskQueue(),
								ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
								SearchAttributes: map[string]interface{}{
									SearchAttributeStack: w.stack,
								},
							},
						),
						RunUpdatePaymentInitiationFromPayment, // nolint:staticcheck // ignore deprecation
						UpdatePaymentInitiationFromPayment{
							Payment: &p,
						},
					).Get(ctx, nil); err != nil {
						errChan <- errors.Wrap(err, "sending events")
					}
				}
			})

			if !IsEventOutboxPatternEnabled(ctx) {
				p := payment

				sendEvents := SendEvents{
					Payment: &p,
				}
				w.runSendEventAsChildWorkflow(ctx, wg, sendEvents, errChan)
			}
		}

		if !IsRunNextTaskOptimizationsEnabled(ctx) {
			for _, payment := range paymentsResponse.Payments {
				p := payment
				payload, err := json.Marshal(p)
				if err != nil {
					errChan <- errors.Wrap(err, "marshalling payment")
				}
				fromPayload := &FromPayload{
					ID:      p.Reference,
					Payload: payload,
				}

				w.runNextTaskAsChildWorkflow(ctx, fetchNextPayments.ConnectorID, nextTasks, wg, fromPayload, errChan)
			}
		} else if len(nextTasks) > 0 {
			// First, we need to get the connector to check if it is scheduled for deletion
			// because if it is, we don't need to run the next tasks
			plugin, err := w.connectors.Get(fetchNextPayments.ConnectorID)
			if err != nil {
				return fmt.Errorf("getting connector: %w", err)
			}

			if !plugin.IsScheduledForDeletion() {
				for _, payment := range paymentsResponse.Payments {
					p := payment

					wg.Add(1)
					workflow.Go(ctx, func(ctx workflow.Context) {
						defer wg.Done()

						payload, err := json.Marshal(p)
						if err != nil {
							errChan <- errors.Wrap(err, "marshalling payment")
							return
						}

						if err := w.runNextTasksV3_1(
							ctx,
							fetchNextPayments.ConnectorID,
							&FromPayload{
								ID:      p.Reference,
								Payload: payload,
							},
							nextTasks,
						); err != nil {
							errChan <- errors.Wrap(err, "running next tasks")
						}
					})
				}
			}
		}

		for _, payment := range paymentsResponse.PaymentsToDelete {
			p := payment

			wg.Add(1)
			workflow.Go(ctx, func(ctx workflow.Context) {
				defer wg.Done()

				if err := activities.StoragePaymentsDeleteFromReference(
					infiniteRetryContext(ctx),
					p.Reference,
					fetchNextPayments.ConnectorID,
				); err != nil {
					errChan <- errors.Wrap(err, "deleting payment")
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

		state.State = paymentsResponse.NewState
		err = activities.StorageStatesStore(
			infiniteRetryContext(ctx),
			*state,
		)
		if err != nil {
			return errors.Wrap(err, "storing state")
		}

		hasMore = paymentsResponse.HasMore

		if w.shouldContinueAsNew(ctx) {
			// If we have lots and lots of payments, sometimes, we need to
			// continue as new to not exceed the maximum history size or length
			// of a workflow.
			return workflow.NewContinueAsNewError(
				ctx,
				RunFetchNextPayments,
				fetchNextPayments,
				nextTasks,
			)
		}
	}

	return nil
}

const RunFetchNextPayments = "FetchPayments"
