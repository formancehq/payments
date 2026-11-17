package connectors

import (
	"encoding/json"
	"io"
	"testing"
	"time"

	logging "github.com/formancehq/go-libs/v5/pkg/observe/log"
	adyenplugin "github.com/formancehq/payments/ce/plugins/adyen"
	stripeplugin "github.com/formancehq/payments/ce/plugins/stripe"
	stripeclient "github.com/formancehq/payments/ce/plugins/stripe/client"
	"github.com/formancehq/payments/internal/connectors/connectorconfig"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v80"
	"go.uber.org/mock/gomock"
)

// Exercise real provider validation and the canonical configuration returned
// by the plugin. The Stripe backend is a local double: no HTTP is performed.
func TestConnectorConfigMergeWithStripeValidation(t *testing.T) {
	for _, tc := range []struct {
		name, patch, want string
		invalid           bool
	}{
		{name: "omitted", patch: `{"name":"new"}`, want: "fake-old"},
		{name: "replaced", patch: `{"apiKey":"fake-new"}`, want: "fake-new"},
		{name: "empty required credential", patch: `{"apiKey":""}`, invalid: true},
		{name: "null required credential", patch: `{"apiKey":null}`, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			merged, err := connectorconfig.Merge(json.RawMessage(`{"name":"old","pollingPeriod":"30m","apiKey":"fake-old"}`), json.RawMessage(tc.patch))
			require.NoError(t, err)
			backend := stripeclient.NewMockBackend(gomock.NewController(t))
			if !tc.invalid {
				backend.EXPECT().Call(gomock.Any(), gomock.Any(), tc.want, nil, gomock.Any()).DoAndReturn(func(_, _, _ string, _ stripe.ParamsContainer, account stripe.LastResponseSetter) error {
					account.(*stripe.Account).ID = "local-account"
					return nil
				})
			}
			plugin, err := stripeplugin.New("local", logging.NewDefaultLogger(io.Discard, false, false, false), merged, backend)
			if tc.invalid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			canonical, err := combineConfigs(models.Config{Name: "new", PollingPeriod: 30 * time.Minute}, plugin.Config())
			require.NoError(t, err)
			var stored struct {
				APIKey string `json:"apiKey"`
			}
			require.NoError(t, json.Unmarshal(canonical, &stored))
			require.True(t, stored.APIKey == tc.want, "credential must survive provider normalization")
		})
	}
}

// Adyen already accepts null for optional strings. Partial updates must keep
// that provider-owned behavior instead of imposing a new cross-provider rule.
// Constructing this plugin only configures its client; it makes no HTTP calls.
func TestConnectorConfigMergeKeepsAdyenOptionalNullSemantics(t *testing.T) {
	logger := logging.NewDefaultLogger(io.Discard, false, false, false)
	full := json.RawMessage(`{"name":"fixture","pollingPeriod":"30m","apiKey":"fake-required","companyID":"local","webhookUsername":"fake-user","webhookPassword":null}`)
	control, err := adyenplugin.New("fixture", logger, full)
	require.NoError(t, err, "the existing provider accepts an optional null field")

	stored := json.RawMessage(`{"name":"fixture","pollingPeriod":"30m","apiKey":"fake-required","companyID":"local","webhookUsername":"fake-user","webhookPassword":"fake-old"}`)
	merged, err := connectorconfig.Merge(stored, json.RawMessage(`{"webhookPassword":null}`))
	require.NoError(t, err, "partial updates must preserve provider validation semantics")
	updated, err := adyenplugin.New("fixture", logger, merged)
	require.NoError(t, err)
	require.True(t, control.Config() == updated.Config(), "omitted credentials and explicit optional null must reach the provider unchanged")
}
