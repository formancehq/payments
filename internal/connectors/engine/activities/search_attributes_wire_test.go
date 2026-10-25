package activities_test

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

// A real SDK client talking to an in-memory WorkflowService captures the wire
// requests without a Temporal deployment or any namespace/admin access.
type searchAttributeService struct {
	workflowservice.UnimplementedWorkflowServiceServer
	starts    []*workflowservice.StartWorkflowExecutionRequest
	schedules []*workflowservice.CreateScheduleRequest
}

func (s *searchAttributeService) GetSystemInfo(context.Context, *workflowservice.GetSystemInfoRequest) (*workflowservice.GetSystemInfoResponse, error) {
	return &workflowservice.GetSystemInfoResponse{}, nil
}
func (s *searchAttributeService) StartWorkflowExecution(_ context.Context, req *workflowservice.StartWorkflowExecutionRequest) (*workflowservice.StartWorkflowExecutionResponse, error) {
	s.starts = append(s.starts, proto.Clone(req).(*workflowservice.StartWorkflowExecutionRequest))
	return &workflowservice.StartWorkflowExecutionResponse{RunId: "run"}, nil
}
func (s *searchAttributeService) CreateSchedule(_ context.Context, req *workflowservice.CreateScheduleRequest) (*workflowservice.CreateScheduleResponse, error) {
	s.schedules = append(s.schedules, proto.Clone(req).(*workflowservice.CreateScheduleRequest))
	return &workflowservice.CreateScheduleResponse{}, nil
}
func searchAttributeClient(t *testing.T) (client.Client, *searchAttributeService) {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	service := &searchAttributeService{}
	workflowservice.RegisterWorkflowServiceServer(server, service)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { server.Stop(); _ = listener.Close() })
	c, err := client.Dial(client.Options{
		HostPort: "passthrough:///search-attribute-test", Namespace: "test", Identity: "test",
		ConnectionOptions: client.ConnectionOptions{DialOptions: []grpc.DialOption{
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		}},
	})
	require.NoError(t, err)
	t.Cleanup(c.Close)
	return c, service
}
func keywordAttributes(values map[string]interface{}) temporal.SearchAttributes {
	updates := make([]temporal.SearchAttributeUpdate, 0, len(values))
	for key, value := range values {
		updates = append(updates, temporal.NewSearchAttributeKeyKeyword(key).ValueSet(value.(string)))
	}
	return temporal.NewSearchAttributes(updates...)
}

// Legacy SDK options are decoded from their old JSON shape. No deprecated
// source field access or lint exclusion is needed to exercise the legacy path.
func decodeLegacyOptions(t *testing.T, values map[string]interface{}, into interface{}) {
	t.Helper()
	data, err := json.Marshal(map[string]interface{}{"SearchAttributes": values})
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, into))
}

func requireKeywordWireEquivalence(t *testing.T, legacy, typed *commonpb.SearchAttributes) {
	t.Helper()
	require.Len(t, typed.GetIndexedFields(), len(legacy.GetIndexedFields()))
	if len(legacy.GetIndexedFields()) == 0 {
		// Legacy non-nil empty maps produce an empty protobuf message; typed empty
		// attributes omit that message. Both carry zero indexed fields.
		return
	}
	normalized := proto.Clone(legacy).(*commonpb.SearchAttributes)
	for key, old := range normalized.IndexedFields {
		got := typed.IndexedFields[key]
		require.NotNil(t, got, key)
		require.Equal(t, old.Data, got.Data, key)
		require.Equal(t, []byte("json/plain"), got.Metadata["encoding"], key)
		require.Equal(t, []byte("Keyword"), got.Metadata["type"], key)
		require.Len(t, got.Metadata, len(old.Metadata)+1, key)
		require.NotContains(t, old.Metadata, "type", key)
		old.Metadata["type"] = []byte("Keyword")
	}
	require.True(t, proto.Equal(normalized, typed), "only Keyword metadata may change")
}

func TestKeywordSearchAttributesWireContract(t *testing.T) {
	c, service := searchAttributeClient(t)
	for _, tc := range []struct {
		name   string
		values map[string]interface{}
	}{
		{"nil", nil}, {"empty", map[string]interface{}{}},
		{"stack-only", map[string]interface{}{"Stack": "stack"}},
		{"strings", map[string]interface{}{"Stack": "", "ConnectorID": "é栈", "PaymentScheduleID": "quote\"\\\n"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			legacy := client.StartWorkflowOptions{}
			decodeLegacyOptions(t, tc.values, &legacy)
			legacy.ID = "workflow"
			legacy.TaskQueue = "queue"
			legacy.WorkflowExecutionTimeout = time.Hour
			legacy.WorkflowRunTimeout = time.Minute
			legacy.WorkflowTaskTimeout = time.Second
			legacy.WorkflowIDReusePolicy = enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY
			typed := client.StartWorkflowOptions{}
			// Copy all non-attribute options, then remove the old serialized field.
			data, err := json.Marshal(legacy)
			require.NoError(t, err)
			var fields map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(data, &fields))
			delete(fields, "SearchAttributes")
			data, err = json.Marshal(fields)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(data, &typed))
			typed.TypedSearchAttributes = keywordAttributes(tc.values)
			_, err = c.ExecuteWorkflow(context.Background(), legacy, "Workflow", "input")
			require.NoError(t, err)
			_, err = c.ExecuteWorkflow(context.Background(), typed, "Workflow", "input")
			require.NoError(t, err)
			old, got := service.starts[len(service.starts)-2], service.starts[len(service.starts)-1]
			requireKeywordWireEquivalence(t, old.SearchAttributes, got.SearchAttributes)
			old.RequestId, got.RequestId = "", ""
			old.SearchAttributes, got.SearchAttributes = nil, nil
			require.True(t, proto.Equal(old, got), "other start options must not change")

			oldSchedule := client.ScheduleOptions{}
			decodeLegacyOptions(t, tc.values, &oldSchedule)
			oldSchedule.ID = "schedule"
			oldSchedule.Spec = client.ScheduleSpec{Intervals: []client.ScheduleIntervalSpec{{Every: time.Minute}}, Jitter: time.Second}
			oldSchedule.Overlap = enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE
			oldSchedule.TriggerImmediately = true
			oldAction := client.ScheduleWorkflowAction{ID: "action", Workflow: "Workflow", TaskQueue: "queue", Args: []interface{}{"input"}}
			oldAction.TypedSearchAttributes = keywordAttributes(tc.values)
			oldSchedule.Action = &oldAction
			newAction := client.ScheduleWorkflowAction{ID: "action", Workflow: "Workflow", TaskQueue: "queue", Args: []interface{}{"input"}, TypedSearchAttributes: keywordAttributes(tc.values)}
			newSchedule := client.ScheduleOptions{ID: oldSchedule.ID, Spec: oldSchedule.Spec, Overlap: oldSchedule.Overlap, TriggerImmediately: true, Action: &newAction, TypedSearchAttributes: keywordAttributes(tc.values)}
			_, err = c.ScheduleClient().Create(context.Background(), oldSchedule)
			require.NoError(t, err)
			_, err = c.ScheduleClient().Create(context.Background(), newSchedule)
			require.NoError(t, err)
			oldReq, newReq := service.schedules[len(service.schedules)-2], service.schedules[len(service.schedules)-1]
			oldWorkflow := oldReq.Schedule.Action.GetStartWorkflow()
			newWorkflow := newReq.Schedule.Action.GetStartWorkflow()
			requireKeywordWireEquivalence(t, oldReq.SearchAttributes, newReq.SearchAttributes)
			require.True(t, proto.Equal(oldWorkflow.SearchAttributes, newWorkflow.SearchAttributes), "action already uses typed attributes before migration")
			oldReq.RequestId, newReq.RequestId = "", ""
			oldReq.SearchAttributes, newReq.SearchAttributes = nil, nil
			oldWorkflow.SearchAttributes, newWorkflow.SearchAttributes = nil, nil
			require.True(t, proto.Equal(oldReq, newReq), "other schedule/action options must not change")
		})
	}
}

// Freeze the current activity boundary, including the asymmetry for non-string
// values. This is a compatibility checkpoint, not a typed schedule migration.
func TestLegacyScheduleActivityDTOAndWireContract(t *testing.T) {
	c, service := searchAttributeClient(t)
	act := activities.New(nil, c, nil, nil, nil, time.Millisecond, 0)
	for _, raw := range []string{
		`{"SearchAttributes":null}`, `{"SearchAttributes":{}}`,
		`{"SearchAttributes":{"Stack":"","ConnectorID":"é栈","PaymentScheduleID":"quote\"\\\n"}}`,
		`{"SearchAttributes":{"Stack":"stack","Number":42,"Boolean":true,"Null":null,"List":["a","b"],"Object":{"k":"v"}}}`,
	} {
		t.Run(raw, func(t *testing.T) {
			// Default converter matches the decoder used for recorded activity inputs.
			raw = `{"ScheduleID":"old-schedule","Interval":{"Every":60000000000},"Action":{"ID":"workflow","Workflow":"Workflow","TaskQueue":"queue"},` + raw[1:]
			payload := &commonpb.Payload{Metadata: map[string][]byte{"encoding": []byte("json/plain")}, Data: []byte(raw)}
			var options activities.ScheduleCreateOptions
			require.NoError(t, converter.GetDefaultDataConverter().FromPayload(payload, &options))
			require.Equal(t, "old-schedule", options.ScheduleID)
			require.Equal(t, time.Minute, options.Interval.Every)
			require.Equal(t, "Workflow", options.Action.Workflow)
			require.NoError(t, act.TemporalScheduleCreate(context.Background(), options))
			req := service.schedules[len(service.schedules)-1]
			outer := req.SearchAttributes.GetIndexedFields()
			action := req.Schedule.Action.GetStartWorkflow().SearchAttributes.GetIndexedFields()
			require.Len(t, outer, len(options.SearchAttributes))
			strings := 0
			for key, value := range options.SearchAttributes {
				expected, err := converter.GetDefaultDataConverter().ToPayload(value)
				require.NoError(t, err)
				require.True(t, proto.Equal(expected, outer[key]), key)
				if _, ok := value.(string); ok {
					strings++
					require.NotNil(t, action[key])
					require.Equal(t, expected.Data, action[key].Data)
					require.Equal(t, []byte("Keyword"), action[key].Metadata["type"])
				} else {
					require.NotContains(t, action, key)
				}
			}
			require.Len(t, action, strings)
		})
	}
}
