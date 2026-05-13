package universal

import (
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/models"
)

// Test-only helpers. The `_test.go` suffix keeps them out of production
// builds while still being exported to *_test.go files in this package's
// black-box test suite (`package universal_test`).

// InjectClient swaps the plugin's HTTP client for a mock so specs can drive
// FetchNext* / Create* / Webhook code paths without standing up an
// httptest.Server for every assertion.
func InjectClient(p *Plugin, c client.Client) {
	p.client = c
}

// InjectFeatures forces the post-install Features (e.g. WebhookSignature)
// so tests can exercise webhook verification without a full Install
// round-trip.
func InjectFeatures(p *Plugin, f client.Features) {
	p.mu.Lock()
	p.features = f
	p.mu.Unlock()
}

// InjectDeclared forces the install-time capability set + the bootstrap
// flag so per-primitive tests can run without going through Install.
func InjectDeclared(p *Plugin, caps []models.Capability) {
	set := newCapabilitySet(caps)
	p.mu.Lock()
	p.declared = set
	p.bootstrapOnInstall = set.has(models.CAPABILITY_FETCH_ORDERS) || set.has(models.CAPABILITY_FETCH_CONVERSIONS)
	p.mu.Unlock()
}
