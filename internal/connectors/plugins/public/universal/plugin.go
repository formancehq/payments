package universal

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

const (
	ProviderName   = "universal"
	MetadataPrefix = "com.universal.spec/"
)

func init() {
	registry.RegisterPlugin(ProviderName, models.PluginTypePSP, func(_ models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{}, PAGE_SIZE)
}

// Plugin is the in-process Universal Connector. The capability set the
// counterparty exposes is discovered at Install via GET /v1/capabilities;
// guard.go enforces it on every per-method call.
//
// accountLookup is injected by the engine right after construction when the
// engine is configured with PluginWithAccountLookup support; orders/conversions
// use it to resolve PSPAccount references at runtime instead of caching them.
type Plugin struct {
	models.Plugin

	name   string
	logger logging.Logger
	client client.Client
	config Config

	mu                sync.RWMutex
	declared          capabilitySet
	features          client.Features
	bootstrapOnInstall bool
	accountLookup     models.AccountLookup
}

// Compile-time assertion that we implement Plugin and the optional upgrade
// interfaces. This catches surface drift if the engine ever extends the
// contract.
var (
	_ models.Plugin                          = &Plugin{}
	_ models.PluginWithAccountLookup         = &Plugin{}
	_ models.PluginWithBootstrapOnInstall    = &Plugin{}
)

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	cfg, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	c := client.New(ProviderName, cfg.Endpoint, cfg.APIKey)

	return &Plugin{
		Plugin: plugins.NewBasePlugin(),
		name:   name,
		logger: logger,
		client: c,
		config: cfg,
	}, nil
}

func (p *Plugin) Name() string                          { return p.name }
func (p *Plugin) Config() models.PluginInternalConfig   { return p.config }
func (p *Plugin) UseAccountLookup(l models.AccountLookup) { p.accountLookup = l }

// BootstrapOnInstall declares that FETCH_ACCOUNTS must run to completion
// before periodic schedules start, but only when ORDERS or CONVERSIONS are
// declared (those primitives reference accounts at runtime). Without that
// constraint we'd race the first poll against an empty accounts table.
func (p *Plugin) BootstrapOnInstall() []models.TaskType {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.bootstrapOnInstall {
		return []models.TaskType{models.TASK_FETCH_ACCOUNTS}
	}
	return nil
}

// Install discovers the counterparty's capability set, validates it against
// the optional CapabilityOverrides, and returns the install-time workflow
// tree. Any failure here aborts the install — the engine will Unload the
// plugin and remove the connector record.
func (p *Plugin) Install(ctx context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	caps, err := p.client.GetCapabilities(ctx)
	if err != nil {
		return models.InstallResponse{}, fmt.Errorf("discovering counterparty capabilities: %w", err)
	}

	declaredRaw, err := parseDeclaredCapabilities(caps.Supported)
	if err != nil {
		return models.InstallResponse{}, fmt.Errorf("parsing /v1/capabilities response: %w", err)
	}
	declared := dedupCapabilities(applyOverrides(declaredRaw, p.config.CapabilityOverridesList()))

	if caps.Features.WebhookSignature == "hmac-sha256" && p.config.WebhookSharedSecret == "" {
		return models.InstallResponse{}, fmt.Errorf("counterparty requires HMAC webhook signatures but webhookSharedSecret is empty")
	}

	// Counterparty may override the canonical Idempotency-Key header
	// name (e.g. legacy "X-Idempotency-Token"). do() always sends the
	// canonical name AND the override so retries dedup either way.
	p.client.SetIdempotencyHeader(caps.Features.IdempotencyHeader)

	set := newCapabilitySet(declared)

	p.mu.Lock()
	p.declared = set
	p.features = caps.Features
	p.bootstrapOnInstall = set.has(models.CAPABILITY_FETCH_ORDERS) || set.has(models.CAPABILITY_FETCH_CONVERSIONS)
	p.mu.Unlock()

	p.logger.WithFields(map[string]any{
		"connector":    p.name,
		"endpoint":     p.config.Endpoint,
		"capabilities": capabilityNames(declared),
		"features":     caps.Features,
		"bootstrap":    p.bootstrapOnInstall,
	}).Info("universal connector installed")

	return models.InstallResponse{Workflow: workflow(set)}, nil
}

func (p *Plugin) Uninstall(_ context.Context, _ models.UninstallRequest) (models.UninstallResponse, error) {
	p.logger.WithField("connector", p.name).Info("universal connector uninstalling")
	return models.UninstallResponse{}, nil
}

// parseDeclaredCapabilities reads the string list returned by the
// counterparty and turns it into typed Capability values. Unknown strings
// fail the install — the contract is versioned, so unknown means the
// counterparty is on a newer spec than us.
func parseDeclaredCapabilities(strs []string) ([]models.Capability, error) {
	out := make([]models.Capability, 0, len(strs))
	for _, s := range strs {
		var c models.Capability
		if err := c.Scan(s); err != nil {
			return nil, fmt.Errorf("unknown capability %q: %w", s, err)
		}
		out = append(out, c)
	}
	return out, nil
}

func capabilityNames(in []models.Capability) []string {
	out := make([]string, len(in))
	for i, c := range in {
		out[i] = c.String()
	}
	return out
}

// declaredSet is the read-side accessor every per-primitive method calls.
// It returns a stable snapshot taken under the read lock so concurrent fetch
// activities never race against a re-install (which doesn't happen today,
// but the lock is cheap and the invariant is worth preserving).
func (p *Plugin) declaredSet() (capabilitySet, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.declared == nil {
		return nil, false
	}
	return p.declared, true
}
