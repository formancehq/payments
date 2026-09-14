package core

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
)

var stackTarget = sdk.TargetSelection{OrganizationID: "org-1", StackID: "stack-1"}
var paymentsV3 = []sdk.ServiceVersion{{Service: sdk.ServicePayments, Version: "3.0.0", Major: 3}}

func TestExecuteGetMapsPathAndEmitsProductJSON(t *testing.T) {
	host := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
		if request.Operation != "v3GetPayment" || request.HTTP.Path != "/v3/payments/pay_1" || request.HTTP.Method != "GET" {
			t.Fatalf("request = %#v", request)
		}
		return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"data":{"id":"pay_1"}}`)}), nil
	})
	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{CommandID: "payments.v3.payments.get", Arguments: []string{"pay_1"}, Target: stackTarget, ServiceVersions: paymentsV3}, host)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	events := host.Events()
	if len(events) != 1 || events[0].Result == nil || string(events[0].Result.Data) != `{"data":{"id":"pay_1"}}` {
		t.Fatalf("events = %#v", events)
	}
	if events[0].Result.OperationID != "payments.v3.payments.get" {
		t.Fatalf("result operation id = %q", events[0].Result.OperationID)
	}
}

type inputHost struct {
	*sdk.MemoryHost
	input []byte
	reads int
}

func (h *inputHost) ReadInput(_ context.Context, handle string) (sdk.InputArtifactChunk, error) {
	if handle != "opaque-input" {
		return sdk.InputArtifactChunk{}, sdk.Failure{Code: string(sdk.FailureInvalidArgument), Message: "bad handle"}
	}
	h.reads++
	if h.reads > 1 {
		return sdk.InputArtifactChunk{}, sdk.Failure{Code: string(sdk.FailureOperationNotPermitted), Message: "consumed handle"}
	}
	return sdk.InputArtifactChunk{Bytes: append([]byte(nil), h.input...), Final: true}, nil
}

func TestExecuteCreateReadsHostArtifactAndForwardsExactBody(t *testing.T) {
	body := []byte(`{"scheme":"card","amount":100,"initialAmount":100,"type":"PAY-IN","createdAt":"2026-09-12T10:00:00Z","connectorID":"con_1","reference":"demo","asset":"USD"}`)
	memory := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
		var decoded map[string]any
		if err := json.Unmarshal(request.HTTP.Body, &decoded); err != nil {
			t.Fatalf("decode generated request: %v", err)
		}
		if request.Operation != "v3CreatePayment" || request.HTTP.Path != "/v3/payments" || request.HTTP.ContentType != "application/json" ||
			decoded["reference"] != "demo" || decoded["connectorID"] != "con_1" || decoded["createdAt"] != "2026-09-12T10:00:00Z" ||
			decoded["type"] != "PAY-IN" || decoded["initialAmount"] != float64(100) || decoded["amount"] != float64(100) || decoded["asset"] != "USD" || decoded["scheme"] != "card" {
			t.Fatalf("request = %#v", request)
		}
		return sdk.NewResponseStream(sdk.Response{Status: 201, ContentType: "application/json", Body: []byte(`{"data":{"id":"pay_2"}}`)}), nil
	})
	host := &inputHost{MemoryHost: memory, input: body}
	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{CommandID: "payments.v3.payments.create", Arguments: []string{"opaque-input"}, Target: stackTarget, ServiceVersions: paymentsV3}, host)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestExecuteInstallConnectorUsesLiveCanonicalProviderForGeneratedUnion(t *testing.T) {
	body := []byte(`{"apiKey":"secret","name":"primary","pageSize":9007199254740993}`)
	calls := 0
	memory := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
		calls++
		switch request.Operation {
		case "v3ListConnectorConfigs":
			return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"data":{"Stripe":{}}}`)}), nil
		case "v3InstallConnector":
			var decoded map[string]any
			if err := json.Unmarshal(request.HTTP.Body, &decoded); err != nil {
				t.Fatalf("decode generated install request: %v", err)
			}
			if request.HTTP.Path != "/v3/connectors/install/stripe" || decoded["provider"] != "Stripe" || decoded["apiKey"] != "secret" || decoded["name"] != "primary" ||
				!strings.Contains(string(request.HTTP.Body), `"pageSize":9007199254740993`) {
				t.Fatalf("generated install request = %#v", request)
			}
			return sdk.NewResponseStream(sdk.Response{Status: 201, ContentType: "application/json", Body: []byte(`{"data":"connector_1"}`)}), nil
		default:
			t.Fatalf("unexpected operation: %s", request.Operation)
			return nil, nil
		}
	})
	host := &inputHost{MemoryHost: memory, input: body}
	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{
		CommandID: "payments.v3.connectors.install", Arguments: []string{"stripe", "opaque-input"}, Target: stackTarget, ServiceVersions: paymentsV3,
	}, host)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestExecuteInstallConnectorRejectsNullConfiguration(t *testing.T) {
	memory := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
		if request.Operation != "v3ListConnectorConfigs" {
			t.Fatalf("unexpected operation after null configuration: %s", request.Operation)
		}
		return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"data":{"Stripe":{}}}`)}), nil
	})
	host := &inputHost{MemoryHost: memory, input: []byte(`null`)}
	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{
		CommandID: "payments.v3.connectors.install", Arguments: []string{"stripe", "opaque-input"}, Target: stackTarget, ServiceVersions: paymentsV3,
	}, host)
	if err == nil {
		t.Fatal("expected null connector configuration rejection")
	}
	if len(host.Events()) != 0 {
		t.Fatalf("events emitted after null connector configuration: %#v", host.Events())
	}
}

func TestExecuteRejectsInvalidGeneratedRequestDTOBeforeHostRequest(t *testing.T) {
	body := []byte(`{"createdAt":{}}`)
	calls := 0
	memory := sdk.NewMemoryHost(func(_ context.Context, _ sdk.Request) (sdk.Responses, error) {
		calls++
		return sdk.NewResponseStream(sdk.Response{Status: 201, ContentType: "application/json", Body: []byte(`{"data":{}}`)}), nil
	})
	host := &inputHost{MemoryHost: memory, input: body}
	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{CommandID: "payments.v3.payments.create", Arguments: []string{"opaque-input"}, Target: stackTarget, ServiceVersions: paymentsV3}, host)
	if err == nil {
		t.Fatal("expected generated request DTO decoding error")
	}
	if calls != 0 {
		t.Fatalf("host requests = %d, want 0", calls)
	}
}

func TestExecuteRejectsNullGeneratedRequestDTOBeforeHostRequest(t *testing.T) {
	calls := 0
	memory := sdk.NewMemoryHost(func(_ context.Context, _ sdk.Request) (sdk.Responses, error) {
		calls++
		return sdk.NewResponseStream(sdk.Response{Status: 201, ContentType: "application/json", Body: []byte(`{"data":{}}`)}), nil
	})
	host := &inputHost{MemoryHost: memory, input: []byte(`null`)}
	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{CommandID: "payments.v3.payments.create", Arguments: []string{"opaque-input"}, Target: stackTarget, ServiceVersions: paymentsV3}, host)
	if err == nil {
		t.Fatal("expected null generated request DTO rejection")
	}
	if calls != 0 {
		t.Fatalf("host requests = %d, want 0", calls)
	}
}

func TestExecuteRejectsInvalidGeneratedResponseDTO(t *testing.T) {
	host := sdk.NewMemoryHost(func(_ context.Context, _ sdk.Request) (sdk.Responses, error) {
		return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"data":"not-a-payment"}`)}), nil
	})
	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{CommandID: "payments.v3.payments.get", Arguments: []string{"pay_1"}, Target: stackTarget, ServiceVersions: paymentsV3}, host)
	if err == nil {
		t.Fatal("expected generated response DTO decoding error")
	}
	if len(host.Events()) != 0 {
		t.Fatalf("events emitted after invalid response DTO: %#v", host.Events())
	}
}

func TestExecuteGeneratedClientDisablesAmbientAuthAndRetries(t *testing.T) {
	t.Setenv("formance_authorization", "Bearer ambient-secret")
	calls := 0
	host := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
		calls++
		if len(request.HTTP.Headers["Authorization"]) != 0 {
			t.Fatalf("ambient authorization reached host request: %#v", request.HTTP.Headers)
		}
		return nil, sdk.Failure{Code: string(sdk.FailureProductHTTPError), Message: "retryable upstream failure", Retryable: true}
	})
	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{CommandID: "payments.v3.payments.get", Arguments: []string{"pay_1"}, Target: stackTarget, ServiceVersions: paymentsV3}, host)
	if err == nil {
		t.Fatal("expected host failure")
	}
	var failure sdk.Failure
	if !errors.As(err, &failure) || !failure.Retryable {
		t.Fatalf("failure = %#v, want retryable host failure", err)
	}
	if calls != 1 {
		t.Fatalf("host requests = %d, want exactly 1", calls)
	}
}

func TestEveryCatalogueOperationUsesExactGeneratedMethod(t *testing.T) {
	commands, specs := Catalogue(), catalogueSpecs()
	seen := map[string]struct{}{}
	for commandIndex, spec := range specs {
		for operationIndex, operation := range spec.operations {
			if _, checked := seen[operation.id]; checked {
				continue
			}
			seen[operation.id] = struct{}{}
			var requests []sdk.Request
			host := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
				requests = append(requests, request)
				return nil, sdk.Failure{Code: string(sdk.FailureProductHTTPError), Message: "stop after generated invocation"}
			})
			arguments := map[string][]string{
				"account-id": {"account_1"}, "bank-account-id": {"bank_1"}, "connector-id": {"connector_1"},
				"schedule-id": {"schedule_1"}, "payment-id": {"payment_1"}, "transfer-id": {"transfer_1"},
				"pool-id": {"pool_1"}, "order-id": {"order_1"}, "conversion-id": {"conversion_1"},
				"task-id": {"task_1"}, "connector": {"Stripe"}, "at": {"2026-09-12T00:00:00Z"},
			}
			flags := map[string][]string{"connector-id": {"connector_1"}}
			body := []byte(`{}`)
			priorResponse := []byte(nil)
			if operation.id == "v3InstallConnector" || operation.id == "v3UpdateConnectorConfig" {
				body = []byte(`{"provider":"Stripe","apiKey":"secret","name":"primary"}`)
				priorResponse = []byte(`{"data":{"Stripe":{}}}`)
			}
			_, _, err := executeOperation(
				context.Background(), host, commands[commandIndex].Operations[operationIndex], operation,
				arguments, flags, "", body, priorResponse,
			)
			if err == nil {
				t.Fatalf("%s unexpectedly completed against rejecting host", operation.id)
			}
			if len(requests) != 1 {
				t.Fatalf("%s generated %d host requests before failure, want 1: %v", operation.id, len(requests), err)
			}
			if got := requests[0].Operation; got != operation.id {
				t.Errorf("%s invoked generated operation %q", operation.id, got)
			}
			if got := requests[0].HTTP.Method; got != operation.method {
				t.Errorf("%s invoked generated HTTP method %q, want %q", operation.id, got, operation.method)
			}
		}
	}
}

func TestExecuteAllPagesFollowsOpaqueCursorAndAggregatesData(t *testing.T) {
	call := 0
	host := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
		call++
		if call == 1 {
			if len(request.HTTP.Query["cursor"]) != 0 {
				t.Fatalf("first cursor = %v", request.HTTP.Query["cursor"])
			}
			return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"cursor":{"data":[{"id":"a"}],"hasMore":true,"next":"opaque-2"}}`)}), nil
		}
		if got := request.HTTP.Query["cursor"]; len(got) != 1 || got[0] != "opaque-2" {
			t.Fatalf("second cursor = %v", got)
		}
		return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"cursor":{"data":[{"id":"b"}],"hasMore":false}}`)}), nil
	})
	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{CommandID: "payments.v3.accounts.list", Target: stackTarget, ServiceVersions: paymentsV3, Continuation: sdk.AllPagesContinuationControl()}, host)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if call != 2 {
		t.Fatalf("calls = %d, want 2", call)
	}
	var items []map[string]any
	if err := json.Unmarshal(host.Events()[0].Result.Data, &items); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if len(items) != 2 || items[0]["id"] != "a" || items[1]["id"] != "b" {
		t.Fatalf("items = %#v", items)
	}
}

func TestExecuteListPreservesGETQueryBody(t *testing.T) {
	query := []byte(`{"sort":"createdAt:desc","wide":9007199254740993}`)
	memory := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
		if request.HTTP.Method != "GET" || string(request.HTTP.Body) != string(query) || request.HTTP.ContentType != "application/json" {
			t.Fatalf("request = %#v", request)
		}
		return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"cursor":{"data":[{"id":"a"}],"hasMore":true,"next":"opaque-next"}}`)}), nil
	})
	host := &inputHost{MemoryHost: memory, input: query}
	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{CommandID: "payments.v3.accounts.list", Flags: []sdk.FlagOccurrence{{Name: "query", Value: "opaque-input"}}, Target: stackTarget, ServiceVersions: paymentsV3}, host)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	result := host.Events()[0].Result
	if string(result.Data) != `[{"id":"a"}]` || result.Page == nil || !result.Page.HasMore || result.Page.NextCursor != "opaque-next" {
		t.Fatalf("single-page result = %#v data=%s", result, result.Data)
	}
}

func TestExecuteUpdateConnectorConfigPreservesWideInteger(t *testing.T) {
	body := []byte(`{"apiKey":"secret","name":"primary","pageSize":9007199254740993}`)
	memory := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
		switch request.Operation {
		case "v3ListConnectorConfigs":
			return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"data":{"Stripe":{}}}`)}), nil
		case "v3UpdateConnectorConfig":
			if !strings.Contains(string(request.HTTP.Body), `"pageSize":9007199254740993`) {
				t.Fatalf("generated update request lost int64 precision: %s", request.HTTP.Body)
			}
			return sdk.NewResponseStream(sdk.Response{Status: 204}), nil
		default:
			t.Fatalf("unexpected operation: %s", request.Operation)
			return nil, nil
		}
	})
	host := &inputHost{MemoryHost: memory, input: body}
	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{
		CommandID: "payments.v3.connectors.update-config", Arguments: []string{"stripe", "opaque-input"},
		Flags: []sdk.FlagOccurrence{{Name: "connector-id", Value: "connector_1"}}, Target: stackTarget, ServiceVersions: paymentsV3,
	}, host)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestExecuteRejectsNullGeneratedQueryBodyBeforeHostRequest(t *testing.T) {
	calls := 0
	memory := sdk.NewMemoryHost(func(_ context.Context, _ sdk.Request) (sdk.Responses, error) {
		calls++
		return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"cursor":{"data":[],"hasMore":false}}`)}), nil
	})
	host := &inputHost{MemoryHost: memory, input: []byte(`null`)}
	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{
		CommandID: "payments.v3.accounts.list", Flags: []sdk.FlagOccurrence{{Name: "query", Value: "opaque-input"}}, Target: stackTarget, ServiceVersions: paymentsV3,
	}, host)
	if err == nil {
		t.Fatal("expected null generated query body rejection")
	}
	if calls != 0 {
		t.Fatalf("host requests = %d, want 0", calls)
	}
}

func TestExecuteAllPagesReadsQueryArtifactOnlyOnce(t *testing.T) {
	query := []byte(`{"sort":"createdAt:desc"}`)
	call := 0
	memory := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
		call++
		if call == 1 {
			if string(request.HTTP.Body) != string(query) || len(request.HTTP.Query["pageSize"]) != 1 || request.HTTP.Query["pageSize"][0] != "25" {
				t.Fatalf("first request = %#v", request.HTTP)
			}
			return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"cursor":{"data":[],"hasMore":true,"next":"next"}}`)}), nil
		}
		if len(request.HTTP.Body) != 0 || len(request.HTTP.Query) != 1 || len(request.HTTP.Query["cursor"]) != 1 || request.HTTP.Query["cursor"][0] != "next" {
			t.Fatalf("continuation request is not cursor-only: %#v", request.HTTP)
		}
		return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"cursor":{"data":[],"hasMore":false}}`)}), nil
	})
	host := &inputHost{MemoryHost: memory, input: query}
	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{CommandID: "payments.v3.accounts.list", Flags: []sdk.FlagOccurrence{{Name: "query", Value: "opaque-input"}, {Name: "page-size", Value: "25"}}, Target: stackTarget, ServiceVersions: paymentsV3, Continuation: sdk.AllPagesContinuationControl()}, host)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if host.reads != 1 {
		t.Fatalf("artifact reads = %d, want 1", host.reads)
	}
}

func TestExecuteAccountBalancesContinuationIsCursorOnly(t *testing.T) {
	call := 0
	host := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
		call++
		if call == 1 {
			if len(request.HTTP.Query["asset"]) != 1 || len(request.HTTP.Query["fromTimestamp"]) != 1 || len(request.HTTP.Query["toTimestamp"]) != 1 || len(request.HTTP.Query["pageSize"]) != 1 {
				t.Fatalf("first account-balances query = %#v", request.HTTP.Query)
			}
			return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"cursor":{"data":[],"hasMore":true,"next":"next"}}`)}), nil
		}
		if len(request.HTTP.Query) != 1 || len(request.HTTP.Query["cursor"]) != 1 || request.HTTP.Query["cursor"][0] != "next" {
			t.Fatalf("continuation account-balances query is not cursor-only: %#v", request.HTTP.Query)
		}
		return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"cursor":{"data":[],"hasMore":false}}`)}), nil
	})
	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{
		CommandID: "payments.v3.accounts.balances", Arguments: []string{"account_1"}, Target: stackTarget, ServiceVersions: paymentsV3,
		Flags: []sdk.FlagOccurrence{
			{Name: "asset", Value: "USD"},
			{Name: "from-timestamp", Value: "2026-09-01T00:00:00Z"},
			{Name: "to-timestamp", Value: "2026-09-12T00:00:00Z"},
			{Name: "page-size", Value: "25"},
		},
		Continuation: sdk.AllPagesContinuationControl(),
	}, host)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if call != 2 {
		t.Fatalf("calls = %d, want 2", call)
	}
}

func TestExecuteRejectsInconsistentPaginationEnvelope(t *testing.T) {
	host := sdk.NewMemoryHost(func(_ context.Context, _ sdk.Request) (sdk.Responses, error) {
		return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"cursor":{"data":[],"hasMore":true}}`)}), nil
	})
	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{CommandID: "payments.v3.accounts.list", Target: stackTarget, ServiceVersions: paymentsV3}, host)
	if err == nil {
		t.Fatal("expected inconsistent pagination error")
	}
	if len(host.Events()) != 0 {
		t.Fatalf("events emitted after invalid page: %#v", host.Events())
	}
}

func TestExecuteMetadataBuildsRequestBody(t *testing.T) {
	host := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
		if request.HTTP.Path != "/v3/payments/pay_1/metadata" || string(request.HTTP.Body) != `{"metadata":{"key":"value","second":"two"}}` {
			t.Fatalf("request = %#v", request)
		}
		return sdk.NewResponseStream(sdk.Response{Status: 204}), nil
	})
	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{CommandID: "payments.v3.payments.set-metadata", Arguments: []string{"pay_1", "key=value", "second=two"}, Target: stackTarget, ServiceVersions: paymentsV3}, host)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	result := host.Events()[0].Result
	if result.OperationID != "payments.v3.payments.set-metadata" || result.Shape != sdk.ResultObject || string(result.Data) != `{}` {
		t.Fatalf("empty mutation result = %#v data=%s", result, result.Data)
	}
}

func TestExecuteGetConnectorConfigRedactsCredentialFields(t *testing.T) {
	host := sdk.NewMemoryHost(func(_ context.Context, _ sdk.Request) (sdk.Responses, error) {
		return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"data":{"provider":"Stripe","apiKey":"secret","name":"primary","pageSize":9007199254740993,"pollingPeriod":"30m"}}`)}), nil
	})
	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{CommandID: "payments.v3.connectors.get-config", Flags: []sdk.FlagOccurrence{{Name: "connector-id", Value: "con_1"}}, Target: stackTarget, ServiceVersions: paymentsV3}, host)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	got := string(host.Events()[0].Result.Data)
	want := `{"data":{"apiKey":"[REDACTED]","name":"primary","pageSize":9007199254740993,"pollingPeriod":"30m","provider":"Stripe"}}`
	if got != want {
		t.Fatalf("redacted output = %s, want %s", got, want)
	}
}

func TestRedactConnectorConfigCoversEveryCredentialKey(t *testing.T) {
	tests := []struct {
		key, provider string
	}{
		{"apiKey", "Stripe"}, {"apiSecret", "Coinbaseprime"}, {"clientSecret", "Tink"},
		{"privateKey", "Fireblocks"}, {"password", "Bankingcircle"}, {"passphrase", "Coinbaseprime"},
		{"accessKey", "Atlar"}, {"secret", "Atlar"}, {"userCertificate", "Bankingcircle"},
		{"userCertificateKey", "Bankingcircle"}, {"configurationToken", "Powens"}, {"stagingToken", "Qonto"},
		{"webhookPassword", "Adyen"}, {"webhookSharedSecret", "Increase"},
	}
	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			encoded := []byte(`{"data":{"provider":"` + test.provider + `","` + test.key + `":"credential"}}`)
			redacted, err := redactConnectorConfig(encoded)
			if err != nil {
				t.Fatalf("redact: %v", err)
			}
			if !strings.Contains(string(redacted), `"`+test.key+`":"[REDACTED]"`) || strings.Contains(string(redacted), "credential") {
				t.Fatalf("%s was not redacted: %s", test.key, redacted)
			}
		})
	}
}
