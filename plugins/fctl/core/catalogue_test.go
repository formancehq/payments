package core

import (
	"reflect"
	"strings"
	"testing"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
	"github.com/formancehq/payments/plugins/fctl/audit"
)

func TestCatalogueCarriesEveryExecutableLegacyCommand(t *testing.T) {
	commands := Catalogue()
	if got, want := len(commands), 44; got != want {
		t.Fatalf("command count = %d, want %d", got, want)
	}
	if err := sdk.ValidateCommands(commands); err != nil {
		t.Fatalf("catalogue is invalid: %v", err)
	}

	want := map[string][]string{
		"payments.v3.accounts.list":            {"v3ListAccounts"},
		"payments.v3.connectors.install":       {"v3ListConnectorConfigs", "v3InstallConnector"},
		"payments.v3.connectors.get-config":    {"v3ListConnectors", "v3GetConnectorConfig"},
		"payments.v3.connectors.update-config": {"v3ListConnectorConfigs", "v3UpdateConnectorConfig"},
	}
	byID := make(map[string]sdk.Command, len(commands))
	for _, command := range commands {
		byID[command.ID] = command
		if len(command.Compatibility) != 1 || command.Compatibility[0].Service != sdk.ServicePayments || len(command.Compatibility[0].Majors) != 1 || command.Compatibility[0].Majors[0] != 3 {
			t.Fatalf("%s has invalid compatibility: %#v", command.ID, command.Compatibility)
		}
	}
	for id, operationIDs := range want {
		command, ok := byID[id]
		if !ok {
			t.Fatalf("missing command %s", id)
		}
		if len(command.Operations) != len(operationIDs) {
			t.Fatalf("%s operation count = %d, want %d", id, len(command.Operations), len(operationIDs))
		}
		for index, operationID := range operationIDs {
			if command.Operations[index].ID != operationID {
				t.Fatalf("%s operation %d = %s, want %s", id, index, command.Operations[index].ID, operationID)
			}
		}
	}
	for id := range byID {
		if strings.Contains(id, "update-status") || strings.Contains(id, "update_status") {
			t.Fatalf("deprecated update-status command was admitted: %s", id)
		}
	}
	for _, baseline := range audit.MappedBaseline() {
		parts := strings.Fields(baseline.Path)
		path := make([]string, 0, len(parts)-1)
		for _, part := range parts[1:] {
			if strings.HasPrefix(part, "<") || strings.HasPrefix(part, "[<") || part == "|-" {
				continue
			}
			path = append(path, part)
		}
		id := "payments.v3." + strings.Join(path, ".")
		command, ok := byID[id]
		if !ok {
			t.Fatalf("baseline %q has no command %q", baseline.Path, id)
		}
		if len(command.Operations) != len(baseline.V3Ops) {
			t.Fatalf("%s operations = %d, want %d", id, len(command.Operations), len(baseline.V3Ops))
		}
		for index, operation := range baseline.V3Ops {
			if command.Operations[index].ID != operation {
				t.Fatalf("%s operation %d = %s, want %s", id, index, command.Operations[index].ID, operation)
			}
		}
	}
}

func TestCatalogueHasExactCommandAndRequestDenominators(t *testing.T) {
	commands := Catalogue()
	if got, want := len(commands), 44; got != want {
		t.Fatalf("command count = %d, want %d", got, want)
	}

	requestCount := 0
	composites := map[string][]string{}
	uniqueOperations := map[string]struct{}{}
	for _, command := range commands {
		requestCount += len(command.Operations)
		operationIDs := make([]string, 0, len(command.Operations))
		for _, operation := range command.Operations {
			operationIDs = append(operationIDs, operation.ID)
			uniqueOperations[operation.ID] = struct{}{}
		}
		if len(operationIDs) > 1 {
			composites[command.ID] = operationIDs
		}
	}
	if requestCount != 47 {
		t.Fatalf("declared request count = %d, want 47", requestCount)
	}
	if got, want := len(uniqueOperations), 44; got != want {
		t.Fatalf("unique operation count = %d, want %d", got, want)
	}
	wantComposites := map[string][]string{
		"payments.v3.connectors.get-config":    {"v3ListConnectors", "v3GetConnectorConfig"},
		"payments.v3.connectors.install":       {"v3ListConnectorConfigs", "v3InstallConnector"},
		"payments.v3.connectors.update-config": {"v3ListConnectorConfigs", "v3UpdateConnectorConfig"},
	}
	if !reflect.DeepEqual(composites, wantComposites) {
		t.Fatalf("composite command requests = %#v, want %#v", composites, wantComposites)
	}
}

func TestCatalogueMarksOnlyConnectorCredentialInputsSensitive(t *testing.T) {
	wantSensitive := map[string]bool{
		"payments.v3.connectors.install":       true,
		"payments.v3.connectors.update-config": true,
	}
	for _, command := range Catalogue() {
		for _, artifact := range command.InputArtifacts {
			if artifact.ArgumentName != "input" {
				continue
			}
			if artifact.Optional {
				t.Errorf("%s required input artifact is optional", command.ID)
			}
			if got, want := artifact.Sensitive, wantSensitive[command.ID]; got != want {
				t.Errorf("%s input sensitive = %t, want %t", command.ID, got, want)
			}
		}
	}
}

func TestCatalogueMarksEveryOptionalQueryArtifactOptional(t *testing.T) {
	queryArtifacts := 0
	for _, command := range Catalogue() {
		for _, artifact := range command.InputArtifacts {
			if artifact.FlagName != "query" {
				continue
			}
			queryArtifacts++
			if !artifact.Optional || artifact.Repeated || artifact.Sensitive {
				t.Errorf("%s query artifact = %#v, want optional single non-sensitive", command.ID, artifact)
			}
		}
	}
	if queryArtifacts != 9 {
		t.Fatalf("query artifacts = %d, want 9", queryArtifacts)
	}
}

func TestCatalogueOperationsMatchTheCurrentOpenAPISpec(t *testing.T) {
	report, err := audit.Build("../../../openapi.yaml")
	if err != nil {
		t.Fatalf("build OpenAPI report: %v", err)
	}
	byID := make(map[string]audit.Record, len(report.V3))
	for _, operation := range report.V3 {
		byID[operation.OperationID] = operation
	}

	for index, spec := range catalogueSpecs() {
		command := Catalogue()[index]
		mutating := false
		for operationIndex, declared := range spec.operations {
			current, ok := byID[declared.id]
			if !ok {
				t.Errorf("%s references operation absent from payments.v3 OpenAPI", declared.id)
				continue
			}
			paginationMatches := declared.paginated == current.Paginated()
			// An intermediate request in a composite command is deliberately
			// single-page: only the final operation may drive command output and
			// continuation. The operation must still be paginated in OpenAPI.
			if operationIndex < len(spec.operations)-1 && current.Paginated() && !declared.paginated {
				paginationMatches = true
			}
			if declared.method != current.Method || declared.path != current.Path || declared.body != current.HasRequestBody() || !paginationMatches {
				t.Errorf("%s declaration = %s %s body=%t paginated=%t, OpenAPI = %s %s body=%t paginated=%t",
					declared.id, declared.method, declared.path, declared.body, declared.paginated,
					current.Method, current.Path, current.HasRequestBody(), current.Paginated())
			}
			wantScopes := current.Scopes
			if !current.HasSecurity {
				wantScopes = []string{}
			}
			if !reflect.DeepEqual(command.Operations[operationIndex].Scopes, wantScopes) {
				t.Errorf("%s scopes = %v, OpenAPI exact scopes = %v", declared.id, command.Operations[operationIndex].Scopes, wantScopes)
			}
			mutating = mutating || current.Mutating()
		}
		wantRisk := sdk.RiskRead
		if mutating {
			wantRisk = sdk.RiskMutation
		}
		if spec.mutation != mutating || command.Risk != wantRisk {
			t.Errorf("%s mutation/risk = %t/%s, OpenAPI-derived = %t/%s", command.ID, spec.mutation, command.Risk, mutating, wantRisk)
		}
	}
}

func TestCatalogueDeclaresExactHostCapabilitiesAndPolicies(t *testing.T) {
	plugin := Plugin{}
	want := sdk.RequiredHostCapabilitiesForCommands(plugin.Commands())
	got := plugin.Metadata().Facets[0].RequiredHostCapabilities
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("required host capabilities = %v, want %v", got, want)
	}
	for _, command := range plugin.Commands() {
		for _, operation := range command.Operations {
			if operation.Service != sdk.ServicePayments || operation.HTTP == nil || operation.HTTP.GeneratedClient == nil {
				t.Fatalf("%s has incomplete HTTP policy: %#v", command.ID, operation)
			}
			undeclared := operation.ID == "v3GetBankAccount" || operation.ID == "v3UpdateBankAccountMetadata" || operation.ID == "v3ForwardBankAccount"
			if undeclared && len(operation.Scopes) != 0 {
				t.Fatalf("%s invented scopes: %v", operation.ID, operation.Scopes)
			}
			if !undeclared && (len(operation.Scopes) != 1 || (operation.Scopes[0] != "payments:read" && operation.Scopes[0] != "payments:write")) {
				t.Fatalf("%s has non-exact scopes: %v", operation.ID, operation.Scopes)
			}
		}
	}
}
