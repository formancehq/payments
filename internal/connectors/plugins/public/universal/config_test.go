package universal_test

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal"
)

// Black-box tests for config parsing — particularly the comma-separated
// capabilityOverrides string the registry's reflection-based OpenAPI
// generator forced us into (the reflector only handles scalar types).

func TestConfig_CapabilityOverridesParsing(t *testing.T) {
	t.Parallel()
	logger := logging.NewDefaultLogger(testWriter{t}, true, false, false)

	cases := []struct {
		name string
		json string
		want []string
	}{
		{"empty", `{"endpoint":"https://x","apiKey":"k"}`, nil},
		{"single", `{"endpoint":"https://x","apiKey":"k","capabilityOverrides":"FETCH_ACCOUNTS"}`, []string{"FETCH_ACCOUNTS"}},
		{"comma-separated with whitespace",
			`{"endpoint":"https://x","apiKey":"k","capabilityOverrides":" FETCH_ACCOUNTS , FETCH_PAYMENTS "}`,
			[]string{"FETCH_ACCOUNTS", "FETCH_PAYMENTS"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			plg, err := universal.New("u", logger, json.RawMessage(tc.json))
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			cfg, ok := plg.Config().(universal.Config)
			if !ok {
				t.Fatalf("Config is not universal.Config (got %T)", plg.Config())
			}
			got := cfg.CapabilityOverridesList()
			if len(got) != len(tc.want) {
				t.Fatalf("len(got)=%d want=%d (%v)", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("idx %d: got=%q want=%q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Logf("%s", p)
	return len(p), nil
}
