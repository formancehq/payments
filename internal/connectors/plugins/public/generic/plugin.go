package generic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/generic/client"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

const ProviderName = "generic"

func init() {
	registry.RegisterPlugin(ProviderName, models.PluginTypePSP, func(_ models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{}, PAGE_SIZE)
}

type Plugin struct {
    models.Plugin

    name   string
    logger logging.Logger

    client client.Client
    config Config
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	client := client.New(ProviderName, config.APIKey, config.Endpoint)

	return &Plugin{
		Plugin: plugins.NewBasePlugin(),

		name:   name,
		logger: logger,
		client: client,
		config: config,
	}, nil
}

func (p *Plugin) Name() string {
    return p.name
}

func (p *Plugin) Config() models.PluginInternalConfig {
    return p.config
}

func (p *Plugin) Install(ctx context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	return models.InstallResponse{
		Workflow: workflow(),
	}, nil
}

func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
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
	return p.fetchExternalAccounts(ctx, req)
}

func (p *Plugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	if p.client == nil {
		return models.FetchNextPaymentsResponse{}, plugins.ErrNotYetInstalled
	}
	return p.fetchNextPayments(ctx, req)
}

func (p *Plugin) CreatePayout(ctx context.Context, req models.CreatePayoutRequest) (models.CreatePayoutResponse, error) {
	if p.client == nil {
		return models.CreatePayoutResponse{}, plugins.ErrNotYetInstalled
	}

	payment, err := p.createPayout(ctx, req.PaymentInitiation)
	if err != nil {
		return models.CreatePayoutResponse{}, err
	}

	// If the payment status is pending or processing, return BOTH Payment AND PollingPayoutID:
	// - Payment: creates the payment record with current status
	// - PollingPayoutID: sets up Temporal schedule to poll for status updates
	if payment.Status == models.PAYMENT_STATUS_PENDING || payment.Status == models.PAYMENT_STATUS_PROCESSING {
		return models.CreatePayoutResponse{
			Payment:         &payment,
			PollingPayoutID: &payment.Reference,
		}, nil
	}

	return models.CreatePayoutResponse{
		Payment: &payment,
	}, nil
}

func (p *Plugin) ReversePayout(ctx context.Context, req models.ReversePayoutRequest) (models.ReversePayoutResponse, error) {
	return models.ReversePayoutResponse{}, fmt.Errorf("payout reversal not supported by generic connector")
}

func (p *Plugin) PollPayoutStatus(ctx context.Context, req models.PollPayoutStatusRequest) (models.PollPayoutStatusResponse, error) {
	if p.client == nil {
		return models.PollPayoutStatusResponse{}, plugins.ErrNotYetInstalled
	}

	payment, err := p.pollPayoutStatus(ctx, req.PayoutID)
	if err != nil {
		return models.PollPayoutStatusResponse{}, err
	}

	// Always return the payment so the workflow can update the record.
	// The workflow checks isPaymentStatusFinal to continue polling for PENDING/PROCESSING.
	return models.PollPayoutStatusResponse{
		Payment: &payment,
	}, nil
}

func (p *Plugin) CreateTransfer(ctx context.Context, req models.CreateTransferRequest) (models.CreateTransferResponse, error) {
	if p.client == nil {
		return models.CreateTransferResponse{}, plugins.ErrNotYetInstalled
	}

	payment, err := p.createTransfer(ctx, req.PaymentInitiation)
	if err != nil {
		return models.CreateTransferResponse{}, err
	}

	// If the payment status is pending or processing, return BOTH Payment AND PollingTransferID:
	// - Payment: creates the payment record with current status
	// - PollingTransferID: sets up Temporal schedule to poll for status updates
	if payment.Status == models.PAYMENT_STATUS_PENDING || payment.Status == models.PAYMENT_STATUS_PROCESSING {
		return models.CreateTransferResponse{
			Payment:           &payment,
			PollingTransferID: &payment.Reference,
		}, nil
	}

	return models.CreateTransferResponse{
		Payment: &payment,
	}, nil
}

func (p *Plugin) ReverseTransfer(ctx context.Context, req models.ReverseTransferRequest) (models.ReverseTransferResponse, error) {
	return models.ReverseTransferResponse{}, fmt.Errorf("transfer reversal not supported by generic connector")
}

func (p *Plugin) PollTransferStatus(ctx context.Context, req models.PollTransferStatusRequest) (models.PollTransferStatusResponse, error) {
	if p.client == nil {
		return models.PollTransferStatusResponse{}, plugins.ErrNotYetInstalled
	}

	payment, err := p.pollTransferStatus(ctx, req.TransferID)
	if err != nil {
		return models.PollTransferStatusResponse{}, err
	}

	// Always return the payment so the workflow can update the record.
	// The workflow checks isPaymentStatusFinal to continue polling for PENDING/PROCESSING.
	return models.PollTransferStatusResponse{
		Payment: &payment,
	}, nil
}

func (p *Plugin) CreateBankAccount(ctx context.Context, req models.CreateBankAccountRequest) (models.CreateBankAccountResponse, error) {
	if p.client == nil {
		return models.CreateBankAccountResponse{}, plugins.ErrNotYetInstalled
	}

	return p.createBankAccount(ctx, req.BankAccount)
}

var _ models.Plugin = &Plugin{}
