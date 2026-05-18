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
	"github.com/pkg/errors"
)

const ProviderName = "universal"

func init() {
	registry.RegisterPlugin(ProviderName, models.PluginTypePSP, func(_ models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{}, PAGE_SIZE)
}

// Plugin is the in-process Universal Connector. The per-install capability
// subset is discovered at Install via GET /v1/capabilities; guard.go
// enforces it on every primitive thereafter.
type Plugin struct {
	models.Plugin

	name   string
	logger logging.Logger
	client client.Client
	config Config

	mu                 sync.RWMutex
	declared           capabilitySet
	features           client.Features
	bootstrapOnInstall bool
	// accountLookup is engine-injected via UseAccountLookup; orders &
	// conversions use it to resolve PSPAccount references at runtime.
	accountLookup models.AccountLookup
}

var (
	_ models.Plugin                       = &Plugin{}
	_ models.PluginWithAccountLookup      = &Plugin{}
	_ models.PluginWithBootstrapOnInstall = &Plugin{}
)

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	cfg, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	return &Plugin{
		Plugin: plugins.NewBasePlugin(),
		name:   name,
		logger: logger,
		client: client.New(ProviderName, cfg.Endpoint, cfg.APIKey),
		config: cfg,
	}, nil
}

func (p *Plugin) Name() string                        { return p.name }
func (p *Plugin) Config() models.PluginInternalConfig { return p.config }

// UseAccountLookup is engine-injected after construction. Lock so a
// concurrent FetchNextBalances during pod startup can never observe a
// torn pointer.
func (p *Plugin) UseAccountLookup(l models.AccountLookup) {
	p.mu.Lock()
	p.accountLookup = l
	p.mu.Unlock()
}

// lookup returns the engine-injected AccountLookup under the read lock.
func (p *Plugin) lookup() models.AccountLookup {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.accountLookup
}

// BootstrapOnInstall forces FETCH_ACCOUNTS to complete before periodic
// schedules start, but only when ORDERS or CONVERSIONS are declared —
// those primitives resolve account references at runtime and would race
// the first poll against an empty accounts table otherwise.
func (p *Plugin) BootstrapOnInstall() []models.TaskType {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.bootstrapOnInstall {
		return []models.TaskType{models.TASK_FETCH_ACCOUNTS}
	}
	return nil
}

// Install discovers the counterparty's capability set, validates it
// against CapabilityOverrides, and returns the workflow tree. Any failure
// here aborts the install — the engine unloads the plugin and removes
// the connector row.
func (p *Plugin) Install(ctx context.Context, _ models.InstallRequest) (models.InstallResponse, error) {
	caps, err := p.client.GetCapabilities(ctx)
	if err != nil {
		return models.InstallResponse{}, fmt.Errorf("discovering counterparty capabilities: %w", err)
	}

	declaredRaw, err := parseDeclaredCapabilities(caps.Supported)
	if err != nil {
		return models.InstallResponse{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}
	if len(declaredRaw) == 0 {
		return models.InstallResponse{}, errors.Wrap(models.ErrInvalidConfig, "counterparty /v1/capabilities returned no supported capabilities")
	}
	narrowed, err := applyOverrides(declaredRaw, p.config.CapabilityOverridesList())
	if err != nil {
		return models.InstallResponse{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}
	declared := dedupCapabilities(narrowed)
	if len(declared) == 0 {
		return models.InstallResponse{}, errors.Wrap(models.ErrInvalidConfig, "capabilityOverrides narrowed declared set to empty")
	}

	if caps.Features.WebhookSignature == "hmac-sha256" && p.config.WebhookSharedSecret == "" {
		return models.InstallResponse{}, errors.Wrap(models.ErrInvalidConfig, "counterparty requires HMAC webhook signatures but webhookSharedSecret is empty")
	}

	// do() always sends the canonical Idempotency-Key AND any
	// counterparty-declared alias (e.g. legacy "X-Idempotency-Token")
	// so retries dedup either way.
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

// Uninstall tears down counterparty-side state owned by this connector.
// Today that's only webhook subscriptions; the engine deletes its own
// schedules and the connector row separately. We always try the
// cleanup so stale subscriptions don't keep firing to a dead endpoint
// (the engine retries Uninstall on partial failure; DELETE is
// idempotent on the counterparty side).
func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	if p.client == nil {
		return models.UninstallResponse{}, plugins.ErrNotYetInstalled
	}
	logger := p.logger.WithFields(map[string]any{
		"connector":    p.name,
		"webhook_subs": len(req.WebhookConfigs),
	})
	if err := p.deleteWebhooks(ctx, req.WebhookConfigs); err != nil {
		logger.Errorf("universal connector uninstall: webhook cleanup failed: %s", err)
		return models.UninstallResponse{}, err
	}
	logger.Info("universal connector uninstalled")
	return models.UninstallResponse{}, nil
}

// parseDeclaredCapabilities rejects unknown strings — the contract is
// versioned, "unknown" means the counterparty advertises a newer spec than
// we know about.
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

// declaredSet returns a stable snapshot under the read lock so concurrent
// fetch activities never race a re-install.
func (p *Plugin) declaredSet() (capabilitySet, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.declared == nil {
		return nil, false
	}
	return p.declared, true
}
