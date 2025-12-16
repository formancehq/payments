package workflow

import (
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

func (w Workflow) runNextTasks(
	ctx workflow.Context,
	config models.Config,
	connector *models.ConnectorIDOnly,
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
				ConnectorID:  connector.ID,
				FromPayload:  fromPayload,
				Periodically: task.Periodically,
			}

			nextWorkflow = RunFetchNextAccounts
			request = req
			capability = models.CAPABILITY_FETCH_ACCOUNTS

		case models.TASK_FETCH_EXTERNAL_ACCOUNTS:
			req := FetchNextExternalAccounts{
				Config:       config,
				ConnectorID:  connector.ID,
				FromPayload:  fromPayload,
				Periodically: task.Periodically,
			}

			nextWorkflow = RunFetchNextExternalAccounts
			request = req
			capability = models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS

		case models.TASK_FETCH_OTHERS:
			req := FetchNextOthers{
				Config:       config,
				ConnectorID:  connector.ID,
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
				ConnectorID:  connector.ID,
				FromPayload:  fromPayload,
				Periodically: task.Periodically,
			}

			nextWorkflow = RunFetchNextPayments
			request = req
			capability = models.CAPABILITY_FETCH_PAYMENTS

		case models.TASK_FETCH_BALANCES:
			req := FetchNextBalances{
				Config:       config,
				ConnectorID:  connector.ID,
				FromPayload:  fromPayload,
				Periodically: task.Periodically,
			}

			nextWorkflow = RunFetchNextBalances
			request = req
			capability = models.CAPABILITY_FETCH_BALANCES

		case models.TASK_CREATE_WEBHOOKS:
			req := CreateWebhooks{
				Config:      config,
				ConnectorID: connector.ID,
				FromPayload: fromPayload,
			}

			nextWorkflow = RunCreateWebhooks
			request = req
			capability = models.CAPABILITY_CREATE_WEBHOOKS

		default:
			return fmt.Errorf("unknown task type: %v", task.TaskType)
		}

		// Schedule next workflow every polling duration
		if task.Periodically {
			// TODO(polo): context
			err := w.scheduleNextWorkflow(
				ctx,
				connector.ID,
				capability,
				task,
				fromPayload,
				nextWorkflow,
				request,
			)
			if err != nil {
				return fmt.Errorf("failed to schedule periodic task: %w", err)
			}

			continue
		}

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

	return nil
}

func (w Workflow) scheduleNextWorkflow(
	ctx workflow.Context,
	connectorID models.ConnectorID,
	capability models.Capability,
	task models.ConnectorTaskTree,
	fromPayload *FromPayload,
	nextWorkflow interface{},
	request interface{},
) error {
	var (
		scheduleID string
	)
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

	config, err := w.connectors.GetConfig(connectorID) // TODO does the manager gets updated in workflows?
	if err != nil {
		return err
	}

	err = activities.TemporalScheduleCreate(
		infiniteRetryContext(ctx),
		activities.ScheduleCreateOptions{
			ScheduleID: scheduleID,
			Jitter:     calculateJitter(config.PollingPeriod),
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
			// schedules are recreated at application start to ensure that any changes made to a connector's workflow take effect
			// allow the workflow from the outdated schedule to finish running before starting a new one with the new workflow
			Overlap:            enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE,
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
	return nil
}

func calculateJitter(pollingPeriod time.Duration) time.Duration {
	maxJitter := time.Minute * 5
	jitter := pollingPeriod / 2
	if jitter <= maxJitter {
		return jitter
	}
	return maxJitter
}

const RunNextTasks = "Run"
