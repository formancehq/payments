package client_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	client "github.com/formancehq/payments/pkg/client"
	"github.com/formancehq/payments/pkg/client/models/components"
)

type localHTTPClient func(*http.Request) (*http.Response, error)

func (f localHTTPClient) Do(r *http.Request) (*http.Response, error) { return f(r) }

// Exercise the generated request serializer: omitted credentials must not be
// synthesized as zero values or defaults when editing configuration metadata.
func TestConnectorConfigPartialUpdateSerialization(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		name := "v3"
		if legacy {
			name = "legacy"
		}
		t.Run(name, func(t *testing.T) {
			patch := map[string]any{"name": "renamed", "providerOptions": map[string]any{"label": "new"}}
			calls := 0
			sdk := client.New("http://local.invalid", client.WithClient(localHTTPClient(func(r *http.Request) (*http.Response, error) {
				calls++
				var got map[string]any
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Fatal("invalid request JSON")
				}
				if !reflect.DeepEqual(got, patch) {
					t.Fatal("SDK changed the supplied partial configuration")
				}
				return &http.Response{StatusCode: http.StatusNoContent, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
			})))
			var err error
			if legacy {
				_, err = sdk.Payments.V1.UpdateConnectorConfigV1(context.Background(), components.ConnectorStripe, "local-id", patch)
			} else {
				_, err = sdk.Payments.V3.V3UpdateConnectorConfig(context.Background(), "local-id", patch)
			}
			if err != nil || calls != 1 {
				t.Fatalf("partial update failed: %v (calls %d)", err, calls)
			}
		})
	}
}

// Read models must accept safe metadata without requiring any stored credential.
func TestConnectorConfigReadWithoutCredentials(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		sdk := client.New("http://local.invalid", client.WithClient(localHTTPClient(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"data":{"name":"local","pollingPeriod":"30m","provider":"Stripe"}}`)), Request: r}, nil
		})))
		var name *string
		if legacy {
			res, err := sdk.Payments.V1.ReadConnectorConfigV1(context.Background(), components.ConnectorStripe, "local-id")
			if err != nil || res.ConnectorConfigResponse == nil {
				t.Fatalf("legacy read failed: %v", err)
			}
			name = res.ConnectorConfigResponse.Data.Name
		} else {
			res, err := sdk.Payments.V3.GetConnectorConfig(context.Background(), "local-id")
			if err != nil || res.V3GetConnectorConfigResponse == nil {
				t.Fatalf("v3 read failed: %v", err)
			}
			name = res.V3GetConnectorConfigResponse.Data.Name
		}
		if name == nil || *name != "local" {
			t.Fatal("read model did not preserve safe metadata")
		}
	}
}
