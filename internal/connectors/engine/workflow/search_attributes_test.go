package workflow

import (
	"fmt"
	"testing"

	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// These tests exercise the existing version gate, not history replay. Real
// pre-migration histories are still required before approving compatibility.
func TestTypedSearchAttributesVersionContract(t *testing.T) {
	connectorID := models.ConnectorID{Reference: uuid.MustParse("11111111-1111-1111-1111-111111111111"), Provider: "test"}
	for _, version := range []workflow.Version{workflow.DefaultVersion, 1} {
		for _, id := range []*models.ConnectorID{nil, &connectorID} {
			t.Run(fmt.Sprintf("version=%d/nil=%t", version, id == nil), func(t *testing.T) {
				var suite testsuite.WorkflowTestSuite
				env := suite.NewTestWorkflowEnvironment()
				env.OnGetVersion(versionFlagConnectorIDSearchAttributeEnabled, workflow.DefaultVersion, 1).Return(version).Once()
				w := Workflow{stack: "栈\"\\"}
				var got temporal.SearchAttributes
				env.ExecuteWorkflow(func(ctx workflow.Context) error {
					got = w.TypedSearchAttributes(ctx, id)
					return nil
				})
				require.NoError(t, env.GetWorkflowError())
				stack, ok := got.GetKeyword(temporal.NewSearchAttributeKeyKeyword(SearchAttributeStack))
				require.True(t, ok)
				require.Equal(t, w.stack, stack)
				value, present := got.GetKeyword(temporal.NewSearchAttributeKeyKeyword(SearchAttributeConnectorID))
				wantConnector := version == 1 && id != nil
				require.Equal(t, wantConnector, present)
				if wantConnector {
					require.Equal(t, connectorID.String(), value)
					require.Equal(t, 2, got.Size())
				} else {
					require.Equal(t, 1, got.Size())
				}
				env.AssertExpectations(t)
			})
		}
	}
}

func TestBootstrapChildSearchAttributes(t *testing.T) {
	connectorID := models.ConnectorID{Reference: uuid.MustParse("11111111-1111-1111-1111-111111111111"), Provider: "test"}
	for _, version := range []workflow.Version{workflow.DefaultVersion, 1} {
		for _, path := range []string{"task", "tasks", "periodic"} {
			t.Run(fmt.Sprintf("%s/version=%d", path, version), func(t *testing.T) {
				var suite testsuite.WorkflowTestSuite
				env := suite.NewTestWorkflowEnvironment()
				w := Workflow{stack: "test"}
				// One call per existing child start; the tasks path starts two children.
				calls := 1
				if path == "tasks" {
					calls = 2
				}
				env.OnGetVersion(versionFlagConnectorIDSearchAttributeEnabled, workflow.DefaultVersion, 1).Return(version).Times(calls)
				env.RegisterWorkflowWithOptions(func(workflow.Context, BootstrapTaskRequest) error { return nil }, workflow.RegisterOptions{Name: RunBootstrapTask})
				env.RegisterWorkflowWithOptions(func(workflow.Context, FetchNextAccounts, []models.ConnectorTaskTree) error { return nil }, workflow.RegisterOptions{Name: RunFetchNextAccounts})
				env.RegisterWorkflowWithOptions(func(workflow.Context, models.ConnectorID, *FromPayload, []models.ConnectorTaskTree) error { return nil }, workflow.RegisterOptions{Name: RunNextTasksV3_1})
				env.OnWorkflow(RunBootstrapTask, mock.Anything, mock.Anything).Maybe().Return(nil)
				env.OnWorkflow(RunFetchNextAccounts, mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)
				env.OnWorkflow(RunNextTasksV3_1, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)
				var names, ids, queues []string
				var attributes []temporal.SearchAttributes
				env.SetOnChildWorkflowStartedListener(func(info *workflow.Info, ctx workflow.Context, _ converter.EncodedValues) {
					names = append(names, info.WorkflowType.Name)
					ids = append(ids, info.WorkflowExecution.ID)
					queues = append(queues, info.TaskQueueName)
					attributes = append(attributes, workflow.GetTypedSearchAttributes(ctx))
				})
				env.ExecuteWorkflow(func(ctx workflow.Context) error {
					switch path {
					case "task":
						return w.runBootstrapTask(ctx, BootstrapTaskRequest{ConnectorID: connectorID, TaskType: models.TASK_FETCH_ACCOUNTS})
					case "tasks":
						return w.bootstrapTasks(ctx, BootstrapTasksRequest{ConnectorID: connectorID, TaskTypes: []models.TaskType{models.TASK_FETCH_ACCOUNTS}})
					default:
						return w.startPeriodicSchedulesForBootstrap(ctx, connectorID, nil)
					}
				})
				require.NoError(t, env.GetWorkflowError())
				wantNames := []string{RunNextTasksV3_1}
				wantIDs := []string{"run-tasks-test-" + connectorID.String()}
				if path == "task" {
					wantNames = []string{RunFetchNextAccounts}
					wantIDs = []string{"bootstrap-fetch-accounts-test-" + connectorID.String()}
				} else if path == "tasks" {
					wantNames = append([]string{RunBootstrapTask}, wantNames...)
					wantIDs = append([]string{fmt.Sprintf("bootstrap-task-test-%s-%d", connectorID.String(), models.TASK_FETCH_ACCOUNTS)}, wantIDs...)
				}
				require.Equal(t, wantNames, names)
				require.Equal(t, wantIDs, ids)
				for i, attrs := range attributes {
					require.Equal(t, "test-default", queues[i])
					stack, ok := attrs.GetKeyword(temporal.NewSearchAttributeKeyKeyword(SearchAttributeStack))
					require.True(t, ok)
					require.Equal(t, "test", stack)
					value, present := attrs.GetKeyword(temporal.NewSearchAttributeKeyKeyword(SearchAttributeConnectorID))
					require.Equal(t, version == 1, present)
					if present {
						require.Equal(t, connectorID.String(), value)
					}
				}
				env.AssertExpectations(t)
			})
		}
	}
}

func TestSendEventChildRemainsStackOnly(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	w := Workflow{stack: "stack"}
	// This callback catches any newly introduced version call on the Stack-only path.
	versionCalls := 0
	env.OnGetVersion(mock.Anything, workflow.DefaultVersion, 1).Maybe().Return(func(string, workflow.Version, workflow.Version) workflow.Version {
		versionCalls++
		return workflow.DefaultVersion
	})
	// Literal persisted workflow name avoids referring to the deprecated exported
	// constant while exercising the retained legacy workflow path.
	env.RegisterWorkflowWithOptions(func(workflow.Context, SendEvents) error { return nil }, workflow.RegisterOptions{Name: "RunSendEvents"})
	var attrs temporal.SearchAttributes
	var queue string
	env.SetOnChildWorkflowStartedListener(func(info *workflow.Info, ctx workflow.Context, _ converter.EncodedValues) {
		queue = info.TaskQueueName
		attrs = workflow.GetTypedSearchAttributes(ctx)
	})
	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		wg := workflow.NewWaitGroup(ctx)
		w.runSendEventAsChildWorkflow(ctx, wg, SendEvents{}, make(chan error, 1))
		wg.Wait(ctx)
		return nil
	})
	require.NoError(t, env.GetWorkflowError())
	require.Zero(t, versionCalls)
	require.Equal(t, "stack-default", queue)
	require.Equal(t, 1, attrs.Size())
	stack, ok := attrs.GetKeyword(temporal.NewSearchAttributeKeyKeyword(SearchAttributeStack))
	require.True(t, ok)
	require.Equal(t, "stack", stack)
	env.AssertExpectations(t)
}
