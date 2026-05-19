package routable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

const ProviderName = "routable"

func init() {
	registry.RegisterPlugin(ProviderName, models.PluginTypePSP, func(_ models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{}, PAGE_SIZE)
}

// Plugin is the dedicated Routable PSP plugin.
type Plugin struct {
	models.Plugin

	name   string
	logger logging.Logger
	client client.Client
	config Config
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	cfg, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}
	return &Plugin{
		Plugin: plugins.NewBasePlugin(),
		name:   name,
		logger: logger,
		client: client.New(ProviderName, cfg.APIKey, cfg.resolvedEndpoint()),
		config: cfg,
	}, nil
}

func (p *Plugin) Name() string {
	return p.name
}

func (p *Plugin) Config() models.PluginInternalConfig {
	return p.config
}

func (p *Plugin) Install(ctx context.Context, _ models.InstallRequest) (models.InstallResponse, error) {
	p.logger.Infof("installing routable connector %q (endpoint=%s, polling=%s)", p.name, p.config.resolvedEndpoint(), p.config.PollingPeriod.Duration())
	// Credential probe: 401/403 surfaces as install error rather than
	// as the first FETCH_ACCOUNTS run failing in the worker.
	if _, err := p.client.ListAccounts(ctx, 1, 1); err != nil {
		return models.InstallResponse{}, fmt.Errorf("verifying routable credentials: %w", err)
	}
	return models.InstallResponse{Workflow: workflow()}, nil
}

func (p *Plugin) Uninstall(_ context.Context, _ models.UninstallRequest) (models.UninstallResponse, error) {
	return models.UninstallResponse{}, nil
}

func (p *Plugin) FetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	if p.client == nil {
		return models.FetchNextAccountsResponse{}, plugins.ErrNotYetInstalled
	}
	return p.fetchNextAccounts(ctx, req)
}

func (p *Plugin) FetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	if p.client == nil {
		return models.FetchNextBalancesResponse{}, plugins.ErrNotYetInstalled
	}
	return p.fetchNextBalances(ctx, req)
}

func (p *Plugin) FetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	if p.client == nil {
		return models.FetchNextExternalAccountsResponse{}, plugins.ErrNotYetInstalled
	}
	return p.fetchNextExternalAccounts(ctx, req)
}

func (p *Plugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	if p.client == nil {
		return models.FetchNextPaymentsResponse{}, plugins.ErrNotYetInstalled
	}
	return p.fetchNextPayments(ctx, req)
}

func (p *Plugin) CreateTransfer(ctx context.Context, req models.CreateTransferRequest) (models.CreateTransferResponse, error) {
	if p.client == nil {
		return models.CreateTransferResponse{}, plugins.ErrNotYetInstalled
	}
	return p.createTransfer(ctx, req)
}

func (p *Plugin) PollTransferStatus(ctx context.Context, req models.PollTransferStatusRequest) (models.PollTransferStatusResponse, error) {
	if p.client == nil {
		return models.PollTransferStatusResponse{}, plugins.ErrNotYetInstalled
	}
	resp, err := p.pollPayableStatus(ctx, req.TransferID)
	if err != nil {
		return models.PollTransferStatusResponse{}, err
	}
	return models.PollTransferStatusResponse(resp), nil
}

func (p *Plugin) CreatePayout(ctx context.Context, req models.CreatePayoutRequest) (models.CreatePayoutResponse, error) {
	if p.client == nil {
		return models.CreatePayoutResponse{}, plugins.ErrNotYetInstalled
	}
	return p.createPayout(ctx, req)
}

func (p *Plugin) PollPayoutStatus(ctx context.Context, req models.PollPayoutStatusRequest) (models.PollPayoutStatusResponse, error) {
	if p.client == nil {
		return models.PollPayoutStatusResponse{}, plugins.ErrNotYetInstalled
	}
	return p.pollPayableStatus(ctx, req.PayoutID)
}

var _ models.Plugin = &Plugin{}
