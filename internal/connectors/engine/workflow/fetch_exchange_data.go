package workflow

import (
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

type FetchExchangeData struct {
	ConnectorID  models.ConnectorID `json:"connectorID"`
	FromPayload  *FromPayload       `json:"fromPayload"`
	Periodically bool               `json:"periodically"`
}

func (w Workflow) runFetchExchangeData(
	ctx workflow.Context,
	fetchExchangeData FetchExchangeData,
	nextTasks []models.ConnectorTaskTree,
) error {
	wg := workflow.NewWaitGroup(ctx)
	var accountFetchErr, orderFetchErr, conversionFetchErr error

	// Fetch accounts and balances
	wg.Add(1)
	workflow.Go(ctx, w.startFetchAccountsForExchange(wg, fetchExchangeData, &accountFetchErr))

	// Fetch orders in parallel
	wg.Add(1)
	workflow.Go(ctx, w.startFetchOrdersForExchange(wg, fetchExchangeData, &orderFetchErr))

	// Fetch conversions in parallel
	wg.Add(1)
	workflow.Go(ctx, w.startFetchConversionsForExchange(wg, fetchExchangeData, &conversionFetchErr))

	wg.Wait(ctx)

	// Check if any of the fetch workflows failed
	if accountFetchErr != nil {
		return accountFetchErr
	}
	if orderFetchErr != nil {
		return orderFetchErr
	}
	if conversionFetchErr != nil {
		return conversionFetchErr
	}

	return nil
}

func (w Workflow) startFetchAccountsForExchange(wg workflow.WaitGroup, fetchExchangeData FetchExchangeData, errPtr *error) func(ctx workflow.Context) {
	return func(ctx workflow.Context) {
		defer wg.Done()

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
			RunFetchNextAccounts,
			FetchNextAccounts{
				ConnectorID:  fetchExchangeData.ConnectorID,
				FromPayload:  fetchExchangeData.FromPayload,
				Periodically: false,
			},
			[]models.ConnectorTaskTree{
				{
					TaskType:     models.TASK_FETCH_BALANCES,
					Name:         "fetch_balances",
					Periodically: false,
					NextTasks:    []models.ConnectorTaskTree{},
				},
			},
		).Get(ctx, nil); err != nil {
			workflow.GetLogger(ctx).Error("failed to fetch accounts", "error", err)
			*errPtr = err
		}
	}
}

func (w Workflow) startFetchOrdersForExchange(wg workflow.WaitGroup, fetchExchangeData FetchExchangeData, errPtr *error) func(ctx workflow.Context) {
	return func(ctx workflow.Context) {
		defer wg.Done()

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
			RunFetchNextOrders,
			FetchNextOrders{
				ConnectorID:  fetchExchangeData.ConnectorID,
				FromPayload:  fetchExchangeData.FromPayload,
				Periodically: false,
			},
			[]models.ConnectorTaskTree{},
		).Get(ctx, nil); err != nil {
			workflow.GetLogger(ctx).Error("failed to fetch orders", "error", err)
			*errPtr = err
		}
	}
}

func (w Workflow) startFetchConversionsForExchange(wg workflow.WaitGroup, fetchExchangeData FetchExchangeData, errPtr *error) func(ctx workflow.Context) {
	return func(ctx workflow.Context) {
		defer wg.Done()

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
			RunFetchNextConversions,
			FetchNextConversions{
				ConnectorID:  fetchExchangeData.ConnectorID,
				FromPayload:  fetchExchangeData.FromPayload,
				Periodically: false,
			},
			[]models.ConnectorTaskTree{},
		).Get(ctx, nil); err != nil {
			workflow.GetLogger(ctx).Error("failed to fetch conversions", "error", err)
			*errPtr = err
		}
	}
}

const RunFetchExchangeData = "FetchExchangeData"
