package universal

import (
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/models"
)

// Test-only helpers — production builds never see them. All three lock
// p.mu so `go test -race` stays clean when specs interleave injections
// with concurrent plugin reads.

// InjectClient swaps the plugin's HTTP client for a mock.
func InjectClient(p *Plugin, c client.Client) {
	p.mu.Lock()
	p.client = c
	p.mu.Unlock()
}

// InjectFeatures forces the post-install Features.
func InjectFeatures(p *Plugin, f client.Features) {
	p.mu.Lock()
	p.features = f
	p.mu.Unlock()
}

// InjectDeclared forces the install-time capability set + the bootstrap
// flag so per-primitive tests skip Install.
func InjectDeclared(p *Plugin, caps []models.Capability) {
	set := newCapabilitySet(caps)
	p.mu.Lock()
	p.declared = set
	p.bootstrapOnInstall = set.has(models.CAPABILITY_FETCH_ORDERS) || set.has(models.CAPABILITY_FETCH_CONVERSIONS)
	p.mu.Unlock()
}
