//go:build fctl_component_guest

package export_formance_fctl_plugin_lifecycle

import (
	"testing"

	pb "github.com/formancehq/fctl-v2-poc/pkg/plugin/protocol/componentbridgev1alpha1"
	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
	"google.golang.org/protobuf/proto"
)

func TestDescribeExportsAllPaymentsCommands(t *testing.T) {
	descriptor, err := pb.DecodeDescriptorEnvelope(Describe())
	if err != nil {
		t.Fatalf("decode descriptor: %v", err)
	}
	if descriptor.Metadata.Name != "payments" || len(descriptor.Commands) != 44 {
		t.Fatalf("descriptor identity/commands = %q/%d", descriptor.Metadata.Name, len(descriptor.Commands))
	}
	queryArtifacts := 0
	for _, command := range descriptor.Commands {
		for _, artifact := range command.InputArtifacts {
			if artifact.FlagName == "query" {
				queryArtifacts++
				if !artifact.Optional {
					t.Errorf("%s query artifact optional=false after guest round trip", command.ID)
				}
			}
		}
	}
	if queryArtifacts != 9 {
		t.Fatalf("query artifacts = %d, want 9", queryArtifacts)
	}
}

func TestPaymentsLifecycleAcceptsAbsentAndPresentOptionalQuery(t *testing.T) {
	start := func(executionID string, flags []*pb.FlagOccurrence) [][]byte {
		return Start(executionID, lifecycleInput(t, executionID, pb.MessageKind_MESSAGE_KIND_START_EXECUTION, &pb.StartPayload{
			Start: &pb.StartPayload_Command{Command: &pb.CommandStart{
				CommandId: "payments.v3.accounts.list", Flags: flags,
				Target:          &pb.TargetCoordinates{OrganizationId: "org-1", StackId: "stack-1"},
				ServiceVersions: []*pb.ServiceVersion{{Service: pb.Service_SERVICE_PAYMENTS, Version: "3.0.0", Major: 3}},
			}},
		}))
	}

	const absentID = "payments-query-absent"
	frames := start(absentID, nil)
	request := lifecycleHostRequest(t, frames)
	if product := request.GetProduct(); product.GetOperationId() != "v3ListAccounts" || len(product.GetHttp().GetBody()) != 0 {
		t.Fatalf("absent query request = %#v", product)
	}
	Close(absentID)

	const presentID = "payments-query-present"
	frames = start(presentID, []*pb.FlagOccurrence{{Name: "query", Value: "opaque-query"}})
	request = lifecycleHostRequest(t, frames)
	if request.GetInputArtifact().GetOpaqueHandle() != "opaque-query" {
		t.Fatalf("present query first request = %#v", request)
	}
	const query = `{"sort":"createdAt:desc"}`
	frames = Resume(presentID, lifecycleInput(t, presentID, pb.MessageKind_MESSAGE_KIND_HOST_RESPONSE, &pb.HostResponsePayload{
		CorrelationId: request.GetCorrelationId(),
		Response: &pb.HostResponsePayload_InputArtifact{InputArtifact: &pb.InputArtifactReadResponse{
			Chunk: []byte(query), Final: true,
		}},
	}))
	request = lifecycleHostRequest(t, frames)
	if product := request.GetProduct(); product.GetOperationId() != "v3ListAccounts" || string(product.GetHttp().GetBody()) != query {
		t.Fatalf("present query product request = %#v", product)
	}
	Close(presentID)
}

func TestPaymentsLifecycleExecutesSensitiveArtifactAndCleansUp(t *testing.T) {
	const executionID = "payments-sensitive-install"
	t.Cleanup(func() { Close(executionID) })

	descriptor, err := pb.DecodeDescriptorEnvelope(Describe())
	if err != nil {
		t.Fatal(err)
	}
	var install *sdk.Command
	for index := range descriptor.Commands {
		if descriptor.Commands[index].ID == "payments.v3.connectors.install" {
			install = &descriptor.Commands[index]
			break
		}
	}
	if install == nil || len(install.InputArtifacts) != 1 || !install.InputArtifacts[0].Sensitive {
		t.Fatalf("install descriptor does not expose one sensitive input artifact: %#v", install)
	}

	frames := Start(executionID, lifecycleInput(t, executionID, pb.MessageKind_MESSAGE_KIND_START_EXECUTION, &pb.StartPayload{
		Start: &pb.StartPayload_Command{Command: &pb.CommandStart{
			CommandId: "payments.v3.connectors.install",
			Arguments: []string{"stripe", "opaque-input"},
			Target:    &pb.TargetCoordinates{OrganizationId: "org-1", StackId: "stack-1"},
			ServiceVersions: []*pb.ServiceVersion{{
				Service: pb.Service_SERVICE_PAYMENTS, Version: "3.0.0", Major: 3,
			}},
		}},
	}))
	request := lifecycleHostRequest(t, frames)
	if request.GetProduct().GetOperationId() != "v3ListConnectorConfigs" {
		t.Fatalf("Start request = %#v, want live connector catalogue lookup", request)
	}

	frames = Resume(executionID, lifecycleInput(t, executionID, pb.MessageKind_MESSAGE_KIND_HOST_RESPONSE, &pb.HostResponsePayload{
		CorrelationId: request.GetCorrelationId(),
		Response: &pb.HostResponsePayload_Product{Product: &pb.ProductResponse{
			Status: 200, ContentType: "application/json", Body: []byte(`{"data":{"Stripe":{}}}`),
		}},
	}))
	request = lifecycleHostRequest(t, frames)
	if request.GetInputArtifact().GetOpaqueHandle() != "opaque-input" {
		t.Fatalf("post-catalogue request = %#v, want sensitive input read", request)
	}

	frames = Resume(executionID, lifecycleInput(t, executionID, pb.MessageKind_MESSAGE_KIND_HOST_RESPONSE, &pb.HostResponsePayload{
		CorrelationId: request.GetCorrelationId(),
		Response: &pb.HostResponsePayload_InputArtifact{InputArtifact: &pb.InputArtifactReadResponse{
			Chunk: []byte(`{"apiKey":"secret","name":"primary"}`), Final: true,
		}},
	}))
	request = lifecycleHostRequest(t, frames)
	if got := request.GetProduct(); got.GetOperationId() != "v3InstallConnector" || got.GetHttp().GetPath() != "/v3/connectors/install/stripe" {
		t.Fatalf("install product request = %#v", got)
	}

	frames = Resume(executionID, lifecycleInput(t, executionID, pb.MessageKind_MESSAGE_KIND_HOST_RESPONSE, &pb.HostResponsePayload{
		CorrelationId: request.GetCorrelationId(),
		Response: &pb.HostResponsePayload_Product{Product: &pb.ProductResponse{
			Status: 201, ContentType: "application/json", Body: []byte(`{"data":"connector_1"}`),
		}},
	}))
	if len(frames) != 2 {
		t.Fatalf("final Resume frames = %d, want result and termination", len(frames))
	}
	lifecycleDecode(t, frames[0], pb.MessageKind_MESSAGE_KIND_EVENT, &pb.EventPayload{})
	var terminal pb.TerminationPayload
	lifecycleDecode(t, frames[1], pb.MessageKind_MESSAGE_KIND_TERMINATION, &terminal)
	if !terminal.GetSuccess() {
		t.Fatalf("terminal = %#v, want success", &terminal)
	}

	Close(executionID)
	frames = Start(executionID, lifecycleInput(t, executionID, pb.MessageKind_MESSAGE_KIND_START_EXECUTION, &pb.StartPayload{
		Start: &pb.StartPayload_Command{Command: &pb.CommandStart{
			CommandId: "payments.v3.payments.get", Arguments: []string{"payment_1"},
			Target:          &pb.TargetCoordinates{OrganizationId: "org-1", StackId: "stack-1"},
			ServiceVersions: []*pb.ServiceVersion{{Service: pb.Service_SERVICE_PAYMENTS, Version: "3.0.0", Major: 3}},
		}},
	}))
	if request = lifecycleHostRequest(t, frames); request.GetProduct().GetOperationId() != "v3GetPayment" {
		t.Fatalf("fresh Start request = %#v", request.GetProduct())
	}
	frames = Cancel(executionID)
	if len(frames) != 1 {
		t.Fatalf("Cancel frames = %d, want one terminal", len(frames))
	}
	terminal.Reset()
	lifecycleDecode(t, frames[0], pb.MessageKind_MESSAGE_KIND_TERMINATION, &terminal)
	if terminal.GetFailure().GetCode() != "canceled" {
		t.Fatalf("Cancel terminal = %#v", &terminal)
	}
	Close(executionID)
}

func lifecycleInput(t *testing.T, executionID string, kind pb.MessageKind, payload proto.Message) []byte {
	t.Helper()
	return lifecycleMarshal(t, &pb.PluginEnvelope{
		ProtocolMajor: pb.ProtocolMajor, FacetKind: pb.FacetKind_FACET_KIND_COMMAND_PROVIDER,
		MessageKind: kind, ExecutionId: executionID, Payload: lifecycleMarshal(t, payload),
	})
}

func lifecycleHostRequest(t *testing.T, frames [][]byte) *pb.HostRequestPayload {
	t.Helper()
	if len(frames) != 1 {
		t.Fatalf("frames = %d, want one host request", len(frames))
	}
	var request pb.HostRequestPayload
	lifecycleDecode(t, frames[0], pb.MessageKind_MESSAGE_KIND_HOST_REQUEST, &request)
	if request.GetCorrelationId() == "" {
		t.Fatal("host request has no correlation ID")
	}
	return &request
}

func lifecycleDecode(t *testing.T, encoded []byte, kind pb.MessageKind, payload proto.Message) {
	t.Helper()
	var envelope pb.PluginEnvelope
	if err := proto.Unmarshal(encoded, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.GetProtocolMajor() != pb.ProtocolMajor || envelope.GetFacetKind() != pb.FacetKind_FACET_KIND_COMMAND_PROVIDER || envelope.GetMessageKind() != kind {
		t.Fatalf("envelope = %#v, want command/%s", &envelope, kind)
	}
	if err := proto.Unmarshal(envelope.GetPayload(), payload); err != nil {
		t.Fatal(err)
	}
}

func lifecycleMarshal(t *testing.T, message proto.Message) []byte {
	t.Helper()
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(message)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}
