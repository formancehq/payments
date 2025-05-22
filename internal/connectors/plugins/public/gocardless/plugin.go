package gocardless

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/gocardless/client"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

const ProviderName = "gocardless"

/*
*
Validation error messages
*/

var (
	ErrMissingAccountNumber = fmt.Errorf("account number is required")
	ErrorMissingCountry     = fmt.Errorf("country is required")
	ErrMissingSwiftBicCode  = fmt.Errorf("swift bic code is required")

	// Metadata errors
	ErrMissingCurrency               = fmt.Errorf("required metadata field %s is missing", client.GocardlessCurrencyMetadataKey)
	ErrNotSupportedCurrency          = fmt.Errorf("invalid currency value for %s metadata field", client.GocardlessCurrencyMetadataKey)
	ErrInvalidCreditorID             = fmt.Errorf("%s ID must start with 'CR'", client.GocardlessCreditorMetadataKey)
	ErrInvalidCustomerID             = fmt.Errorf("%s ID must start with 'CU'", client.GocardlessCustomerMetadataKey)
	ErrCreditorAndCustomerIDProvided = fmt.Errorf("you must provide either %s or %s metadata field but not both", client.GocardlessCustomerMetadataKey, client.GocardlessCreditorMetadataKey)

	ErrMissingSwiftCode    = fmt.Errorf("field swiftBicCode is required for US bank accounts")
	ErrMissingAccountType  = fmt.Errorf("required metadata field %s is missing", client.GocardlessAccountTypeMetadataKey)
	ErrAccountTypeProvided = fmt.Errorf("metadata field %s is not required for non USD bank accounts", client.GocardlessAccountTypeMetadataKey)
	ErrInvalidAccountType  = fmt.Errorf("metadata field %s must be checking or savings", client.GocardlessAccountTypeMetadataKey)
)

func init() {
	registry.RegisterPlugin(ProviderName, func(_ models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		return New(name, logger, rm)
	}, Capabilities, Config{})
}

type Plugin struct {
	name   string
	logger logging.Logger

	client client.Client
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	shouldFetchMandate := config.ShouldFetchMandate == "true"

	client, err := client.New(ProviderName, config.Endpoint, config.AccessToken, shouldFetchMandate)

	if err != nil {
		return nil, err
	}

	return &Plugin{
		name:   name,
		logger: logger,

		client: client,
	}, nil
}

func (p *Plugin) Name() string {
	return p.name
}

func (p *Plugin) Install(_ context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	return models.InstallResponse{
		Workflow: Workflow(),
	}, nil
}

func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	return models.UninstallResponse{}, nil
}

func (p *Plugin) FetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	if p.client == nil {
		return models.FetchNextAccountsResponse{}, plugins.ErrNotYetInstalled
	}
	return models.FetchNextAccountsResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) FetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	if p.client == nil {
		return models.FetchNextBalancesResponse{}, plugins.ErrNotYetInstalled
	}
	return models.FetchNextBalancesResponse{}, plugins.ErrNotImplemented
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

func (p *Plugin) FetchNextOthers(ctx context.Context, req models.FetchNextOthersRequest) (models.FetchNextOthersResponse, error) {

	if p.client == nil {
		return models.FetchNextOthersResponse{}, plugins.ErrNotYetInstalled
	}

	return p.fetchNextUsers(ctx, req)

}

func (p *Plugin) CreateBankAccount(ctx context.Context, req models.CreateBankAccountRequest) (models.CreateBankAccountResponse, error) {
	if p.client == nil {
		return models.CreateBankAccountResponse{}, plugins.ErrNotYetInstalled
	}

	return p.createBankAccount(ctx, req.BankAccount)
}

func (p *Plugin) CreateTransfer(ctx context.Context, req models.CreateTransferRequest) (models.CreateTransferResponse, error) {
	if p.client == nil {
		return models.CreateTransferResponse{}, plugins.ErrNotYetInstalled
	}

	return models.CreateTransferResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) ReverseTransfer(ctx context.Context, req models.ReverseTransferRequest) (models.ReverseTransferResponse, error) {
	if p.client == nil {
		return models.ReverseTransferResponse{}, plugins.ErrNotYetInstalled
	}

	return models.ReverseTransferResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) PollTransferStatus(ctx context.Context, req models.PollTransferStatusRequest) (models.PollTransferStatusResponse, error) {
	if p.client == nil {
		return models.PollTransferStatusResponse{}, plugins.ErrNotYetInstalled
	}

	return models.PollTransferStatusResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) CreatePayout(ctx context.Context, req models.CreatePayoutRequest) (models.CreatePayoutResponse, error) {
	if p.client == nil {
		return models.CreatePayoutResponse{}, plugins.ErrNotYetInstalled
	}

	return models.CreatePayoutResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) ReversePayout(ctx context.Context, req models.ReversePayoutRequest) (models.ReversePayoutResponse, error) {
	if p.client == nil {
		return models.ReversePayoutResponse{}, plugins.ErrNotYetInstalled
	}

	return models.ReversePayoutResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) PollPayoutStatus(ctx context.Context, req models.PollPayoutStatusRequest) (models.PollPayoutStatusResponse, error) {
	if p.client == nil {
		return models.PollPayoutStatusResponse{}, plugins.ErrNotYetInstalled
	}

	return models.PollPayoutStatusResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) CreateWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	if p.client == nil {
		return models.CreateWebhooksResponse{}, plugins.ErrNotYetInstalled
	}

	return models.CreateWebhooksResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) VerifyWebhook(ctx context.Context, req models.VerifyWebhookRequest) (models.VerifyWebhookResponse, error) {
	if p.client == nil {
		return models.VerifyWebhookResponse{}, plugins.ErrNotYetInstalled
	}
	return models.VerifyWebhookResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) TranslateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	if p.client == nil {
		return models.TranslateWebhookResponse{}, plugins.ErrNotYetInstalled
	}

	return models.TranslateWebhookResponse{}, plugins.ErrNotImplemented
}

var _ models.Plugin = &Plugin{}
