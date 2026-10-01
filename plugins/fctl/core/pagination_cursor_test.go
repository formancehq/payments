package core

import (
	"context"
	"sort"
	"testing"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
)

// paginatedCommandIDs is the exact set of commands whose final operation is a
// cursor listing, and therefore the exact set that must expose a resume point.
func paginatedCommandIDs() []string {
	return []string{
		"payments.v3.accounts.balances",
		"payments.v3.accounts.list",
		"payments.v3.bank_accounts.list",
		"payments.v3.connectors.list",
		"payments.v3.connectors.schedules.instances.list",
		"payments.v3.connectors.schedules.list",
		"payments.v3.conversions.list",
		"payments.v3.orders.list",
		"payments.v3.payments.list",
		"payments.v3.pools.list",
		"payments.v3.transfer_initiation.list",
	}
}

func TestEveryPaginatedCommandDeclaresAnOptionalCursorFlag(t *testing.T) {
	want := paginatedCommandIDs()
	got := make([]string, 0, len(want))
	for _, command := range Catalogue() {
		if !command.Pagination.Supported {
			for _, flag := range command.Flags {
				if flag.Name == "cursor" {
					t.Errorf("%s is not paginated but declares a cursor flag", command.ID)
				}
			}
			continue
		}
		got = append(got, command.ID)
		found := false
		for _, flag := range command.Flags {
			if flag.Name != "cursor" {
				continue
			}
			found = true
			if flag.Required {
				t.Errorf("%s cursor flag is required", command.ID)
			}
			if flag.Type != sdk.FlagString {
				t.Errorf("%s cursor flag type = %s, want %s", command.ID, flag.Type, sdk.FlagString)
			}
		}
		if !found {
			t.Errorf("%s emits a next cursor but declares no cursor flag", command.ID)
		}
	}
	sort.Strings(got)
	if len(got) != len(want) {
		t.Fatalf("paginated commands = %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("paginated commands = %v, want %v", got, want)
		}
	}
}

func TestSinglePageExecutionForwardsTheSuppliedCursorAlone(t *testing.T) {
	calls := 0
	host := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
		calls++
		if len(request.HTTP.Query) != 1 || len(request.HTTP.Query["cursor"]) != 1 || request.HTTP.Query["cursor"][0] != "page-2" {
			t.Fatalf("resumed query is not cursor-only: %#v", request.HTTP.Query)
		}
		if len(request.HTTP.Body) != 0 {
			t.Fatalf("resumed request carried a body: %s", request.HTTP.Body)
		}
		return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json",
			Body: []byte(`{"cursor":{"data":[{"id":"acc_2"}],"hasMore":true,"next":"page-3"}}`)}), nil
	})
	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{
		CommandID: "payments.v3.accounts.list", Target: stackTarget, ServiceVersions: paymentsV3,
		Flags: []sdk.FlagOccurrence{{Name: "cursor", Value: "page-2"}},
	}, host)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if calls != 1 {
		t.Fatalf("host requests = %d, want 1", calls)
	}
	result := host.Events()[0].Result
	if result.Shape != sdk.ResultCollection || string(result.Data) != `[{"id":"acc_2"}]` {
		t.Fatalf("resumed result = %#v data=%s", result, result.Data)
	}
	if result.Page == nil || result.Page.NextCursor != "page-3" || !result.Page.HasMore {
		t.Fatalf("resumed page info = %#v", result.Page)
	}
}

func TestAllPagesExecutionResumesFromTheSuppliedCursor(t *testing.T) {
	calls := 0
	host := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
		calls++
		want := map[int]string{1: "page-2", 2: "page-3"}
		if len(request.HTTP.Query["cursor"]) != 1 || request.HTTP.Query["cursor"][0] != want[calls] {
			t.Fatalf("call %d query = %#v, want cursor %q", calls, request.HTTP.Query, want[calls])
		}
		if calls == 1 {
			return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json",
				Body: []byte(`{"cursor":{"data":[{"id":"acc_2"}],"hasMore":true,"next":"page-3"}}`)}), nil
		}
		return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json",
			Body: []byte(`{"cursor":{"data":[{"id":"acc_3"}],"hasMore":false}}`)}), nil
	})
	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{
		CommandID: "payments.v3.accounts.list", Target: stackTarget, ServiceVersions: paymentsV3,
		Flags:        []sdk.FlagOccurrence{{Name: "cursor", Value: "page-2"}},
		Continuation: sdk.AllPagesContinuationControl(),
	}, host)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if calls != 2 {
		t.Fatalf("host requests = %d, want 2", calls)
	}
	if got := string(host.Events()[0].Result.Data); got != `[{"id":"acc_2"},{"id":"acc_3"}]` {
		t.Fatalf("resumed all-pages data = %s", got)
	}
}

func TestCursorIsRejectedAlongsideAnyOtherFilteringFlag(t *testing.T) {
	tests := []struct {
		name      string
		commandID string
		arguments []string
		flags     []sdk.FlagOccurrence
	}{
		{"page-size", "payments.v3.accounts.list", nil, []sdk.FlagOccurrence{
			{Name: "cursor", Value: "page-2"}, {Name: "page-size", Value: "25"},
		}},
		{"query", "payments.v3.accounts.list", nil, []sdk.FlagOccurrence{
			{Name: "cursor", Value: "page-2"}, {Name: "query", Value: "opaque-input"},
		}},
		{"asset", "payments.v3.accounts.balances", []string{"account_1"}, []sdk.FlagOccurrence{
			{Name: "cursor", Value: "page-2"}, {Name: "asset", Value: "USD"},
		}},
		{"from-timestamp", "payments.v3.accounts.balances", []string{"account_1"}, []sdk.FlagOccurrence{
			{Name: "cursor", Value: "page-2"}, {Name: "from-timestamp", Value: "2026-09-01T00:00:00Z"},
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			memory := sdk.NewMemoryHost(func(_ context.Context, _ sdk.Request) (sdk.Responses, error) {
				calls++
				return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json",
					Body: []byte(`{"cursor":{"data":[],"hasMore":false}}`)}), nil
			})
			host := &inputHost{MemoryHost: memory, input: []byte(`{"id":"acc_1"}`)}
			err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{
				CommandID: test.commandID, Arguments: test.arguments, Flags: test.flags,
				Target: stackTarget, ServiceVersions: paymentsV3,
			}, host)
			if err == nil {
				t.Fatal("expected cursor/filter combination to be rejected")
			}
			if calls != 0 {
				t.Fatalf("host requests = %d, want 0", calls)
			}
			if len(host.Events()) != 0 {
				t.Fatalf("events emitted after rejection: %#v", host.Events())
			}
		})
	}
}
