package core

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	paymentscomponents "github.com/formancehq/payments/pkg/client/models/components"
)

// acknowledgedNonCredentialConnectorFields is the reviewed half of the
// redact-or-acknowledge contract. Every JSON property reachable from the
// generated V3ConnectorConfig union must appear either here or in
// credentialFieldNames; a regenerated client that introduces an unreviewed
// property fails closed instead of forwarding it to the host.
var acknowledgedNonCredentialConnectorFields = map[string]string{
	"actingTeamMember":      "Routable team-member UUID acting on a payable; an identifier, not a credential",
	"authEndpoint":          "provider authentication URL",
	"authorizationEndpoint": "provider authorization URL",
	"baseURL":               "provider base URL",
	"baseUrl":               "provider base URL (generated spelling variant)",
	"clientID":              "public half of a client-credentials pair; the secret half is clientSecret",
	"companyID":             "Adyen merchant company identifier",
	"directory":             "Dummypay filesystem directory the connector reads",
	"domain":                "Powens per-tenant domain used to derive the endpoint",
	"endpoint":              "provider endpoint URL",
	"isSandbox":             "sandbox selector flag",
	"linkFlowError":         "test-only flag forcing a link-flow error",
	"liveEndpointPrefix":    "Adyen live endpoint prefix",
	"loginID":               "public login identifier; the secret half is apiKey",
	"maxConnectionsPerLink": "numeric connection ceiling",
	"name":                  "operator-chosen connector name",
	"pageSize":              "numeric fetch page size",
	"pollingPeriod":         "ISO-8601 polling period",
	"portfolioId":           "Coinbase Prime portfolio identifier",
	"provider":              "connector provider discriminator",
	"updateLinkFlowError":   "test-only flag forcing an update-link-flow error",
	"username":              "public half of a username/password pair; the secret half is password",
	"webhookPublicKey":      "public verification key for inbound webhooks",
	"webhookUsername":       "public half of a webhook basic-auth pair; the secret half is webhookPassword",
}

// generatedConnectorConfigFields returns every JSON property name declared by
// any member of the generated connector-configuration union.
func generatedConnectorConfigFields(t *testing.T) map[string][]string {
	t.Helper()
	union := reflect.TypeOf(paymentscomponents.V3ConnectorConfig{})
	owners := map[string][]string{}
	for index := 0; index < union.NumField(); index++ {
		member := union.Field(index).Type
		for member.Kind() == reflect.Pointer {
			member = member.Elem()
		}
		if member.Kind() != reflect.Struct {
			continue
		}
		for field := 0; field < member.NumField(); field++ {
			name := strings.Split(member.Field(field).Tag.Get("json"), ",")[0]
			if name == "" || name == "-" {
				continue
			}
			owners[name] = append(owners[name], member.Name())
		}
	}
	if len(owners) == 0 {
		t.Fatal("generated connector configuration union exposes no JSON properties")
	}
	return owners
}

func TestEveryGeneratedConnectorConfigFieldIsRedactedOrAcknowledged(t *testing.T) {
	unreviewed := make([]string, 0)
	for name := range generatedConnectorConfigFields(t) {
		_, redacted := credentialFieldNames[name]
		_, acknowledged := acknowledgedNonCredentialConnectorFields[name]
		switch {
		case redacted && acknowledged:
			t.Errorf("connector config field %q is both redacted and acknowledged non-credential", name)
		case !redacted && !acknowledged:
			unreviewed = append(unreviewed, name)
		}
	}
	sort.Strings(unreviewed)
	if len(unreviewed) != 0 {
		t.Fatalf("unreviewed generated connector configuration fields %v: add each to credentialFieldNames "+
			"(redact) or to acknowledgedNonCredentialConnectorFields (with the evidence that it carries no credential)", unreviewed)
	}
}

func TestRedactionDenylistContainsNoDeadFieldName(t *testing.T) {
	declared := generatedConnectorConfigFields(t)
	dead := make([]string, 0)
	for name := range credentialFieldNames {
		if _, exists := declared[name]; !exists {
			dead = append(dead, name)
		}
	}
	sort.Strings(dead)
	if len(dead) != 0 {
		t.Fatalf("credentialFieldNames lists field names absent from the generated union: %v", dead)
	}
	deadAcknowledged := make([]string, 0)
	for name := range acknowledgedNonCredentialConnectorFields {
		if _, exists := declared[name]; !exists {
			deadAcknowledged = append(deadAcknowledged, name)
		}
	}
	sort.Strings(deadAcknowledged)
	if len(deadAcknowledged) != 0 {
		t.Fatalf("acknowledgedNonCredentialConnectorFields lists field names absent from the generated union: %v", deadAcknowledged)
	}
}

// TestRedactionCoversEveryDeclaringProviderOfEveryCredentialField proves the
// behaviour rather than the list: for every union member that declares a
// credential property, a real response for that provider comes back redacted.
func TestRedactionCoversEveryDeclaringProviderOfEveryCredentialField(t *testing.T) {
	union := reflect.TypeOf(paymentscomponents.V3ConnectorConfig{})
	checked := 0
	for index := 0; index < union.NumField(); index++ {
		member := union.Field(index).Type
		for member.Kind() == reflect.Pointer {
			member = member.Elem()
		}
		if member.Kind() != reflect.Struct {
			continue
		}
		provider := strings.TrimSuffix(strings.TrimPrefix(member.Name(), "V3"), "Config")
		for field := 0; field < member.NumField(); field++ {
			name := strings.Split(member.Field(field).Tag.Get("json"), ",")[0]
			if _, sensitive := credentialFieldNames[name]; !sensitive {
				continue
			}
			checked++
			encoded := []byte(`{"data":{"provider":"` + provider + `","` + name + `":"credential-value"}}`)
			redacted, err := redactConnectorConfig(encoded)
			if err != nil {
				t.Errorf("%s.%s: redact: %v", provider, name, err)
				continue
			}
			var decoded struct {
				Data map[string]json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(redacted, &decoded); err != nil {
				t.Errorf("%s.%s: decode redacted config: %v", provider, name, err)
				continue
			}
			if strings.Contains(string(redacted), "credential-value") {
				t.Errorf("%s.%s survived redaction: %s", provider, name, redacted)
			}
			if string(decoded.Data[name]) != `"[REDACTED]"` {
				t.Errorf("%s.%s = %s, want \"[REDACTED]\"", provider, name, decoded.Data[name])
			}
		}
	}
	if checked == 0 {
		t.Fatal("no credential-bearing provider configuration was exercised")
	}
}
