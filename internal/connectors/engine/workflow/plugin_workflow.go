package workflow

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

func (w Workflow) run(
	ctx workflow.Context,
	config models.Config,
	connectorID models.ConnectorID,
	fromPayload *FromPayload,
	taskTree []models.ConnectorTaskTree,
) error {
	var nextWorkflow interface{}
	var request interface{}
	var capability models.Capability
	for _, task := range taskTree {
		switch task.TaskType {
		case models.TASK_FETCH_ACCOUNTS:
			req := FetchNextAccounts{
				Config:       config,
				ConnectorID:  connectorID,
				FromPayload:  fromPayload,
				Periodically: task.Periodically,
			}

			nextWorkflow = RunFetchNextAccounts
			request = req
			capability = models.CAPABILITY_FETCH_ACCOUNTS

		case models.TASK_FETCH_EXTERNAL_ACCOUNTS:
			req := FetchNextExternalAccounts{
				Config:       config,
				ConnectorID:  connectorID,
				FromPayload:  fromPayload,
				Periodically: task.Periodically,
			}

			nextWorkflow = RunFetchNextExternalAccounts
			request = req
			capability = models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS

		case models.TASK_FETCH_OTHERS:
			req := FetchNextOthers{
				Config:       config,
				ConnectorID:  connectorID,
				Name:         task.Name,
				FromPayload:  fromPayload,
				Periodically: task.Periodically,
			}

			nextWorkflow = RunFetchNextOthers
			request = req
			capability = models.CAPABILITY_FETCH_OTHERS

		case models.TASK_FETCH_PAYMENTS:
			req := FetchNextPayments{
				Config:       config,
				ConnectorID:  connectorID,
				FromPayload:  fromPayload,
				Periodically: task.Periodically,
			}

			nextWorkflow = RunFetchNextPayments
			request = req
			capability = models.CAPABILITY_FETCH_PAYMENTS

		case models.TASK_FETCH_BALANCES:
			req := FetchNextBalances{
				Config:       config,
				ConnectorID:  connectorID,
				FromPayload:  fromPayload,
				Periodically: task.Periodically,
			}

			nextWorkflow = RunFetchNextBalances
			request = req
			capability = models.CAPABILITY_FETCH_BALANCES

		case models.TASK_CREATE_WEBHOOKS:
			req := CreateWebhooks{
				Config:      config,
				ConnectorID: connectorID,
				FromPayload: fromPayload,
			}

			nextWorkflow = RunCreateWebhooks
			request = req
			capability = models.CAPABILITY_CREATE_WEBHOOKS

		default:
			return fmt.Errorf("unknown task type: %v", task.TaskType)
		}

		if task.Periodically {
			// Schedule next workflow every polling duration
			// TODO(polo): context
			var scheduleID string
			if fromPayload == nil {
				scheduleID = fmt.Sprintf("%s-%s-%s", w.stack, connectorID.String(), capability.String())
			} else {
				scheduleID = fmt.Sprintf("%s-%s-%s-%s", w.stack, connectorID.String(), capability.String(), fromPayload.ID)
			}

			err := activities.StorageSchedulesStore(
				infiniteRetryContext(ctx),
				models.Schedule{
					ID:          scheduleID,
					ConnectorID: connectorID,
					CreatedAt:   workflow.Now(ctx).UTC(),
				})
			if err != nil {
				return err
			}

			err = activities.TemporalScheduleCreate(
				infiniteRetryContext(ctx),
				activities.ScheduleCreateOptions{
					ScheduleID: scheduleID,
					Jitter:     config.PollingPeriod / 2,
					Interval: client.ScheduleIntervalSpec{
						Every: config.PollingPeriod,
					},
					Action: client.ScheduleWorkflowAction{
						// Use the same ID as the schedule ID, so we can identify the workflows running.
						// This is useful for debugging purposes.
						ID:       scheduleID,
						Workflow: nextWorkflow,
						Args: []interface{}{
							request,
							task.NextTasks,
						},
						TaskQueue: w.getDefaultTaskQueue(),
					},
					Overlap:            enums.SCHEDULE_OVERLAP_POLICY_SKIP,
					TriggerImmediately: true,
					SearchAttributes: map[string]any{
						SearchAttributeScheduleID: scheduleID,
						SearchAttributeStack:      w.stack,
					},
				},
			)
			if err != nil {
				return err
			}

		} else {
			// Run next workflow immediately
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
				nextWorkflow,
				request,
				task.NextTasks,
			).GetChildWorkflowExecution().Get(ctx, nil); err != nil {
				return errors.Wrap(err, "running next workflow")
			}
		}
	}
	return nil
}

const Run = "Run"
