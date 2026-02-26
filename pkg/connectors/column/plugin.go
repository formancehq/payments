package column

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/formancehq/payments/pkg/connectors/column/client"
	"github.com/formancehq/payments/pkg/registry"
)

const ProviderName = "column"

func init() {
	registry.RegisterPlugin(ProviderName, connector.PluginTypePSP, func(connectorID connector.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (connector.Plugin, error) {
		return New(connectorID, name, logger, rm)
	}, capabilities, Config{}, PAGE_SIZE)
}

/*
*
Validation error messages
*/
var (
	ErrMissingAmount         = errors.New("required field amount must be provided")
	ErrMissingAsset          = errors.New("required field asset must be provided")
	ErrAccountNumberRequired = errors.New("required field accountNumber must be provided")

	ErrMissingSourceAccount           = errors.New("required field sourceAccount must be provided")
	ErrMissingSourceAccountName       = errors.New("required sourceAccount field name must be provided")
	ErrSourceAccountReferenceRequired = errors.New("required sourceAccount field reference must be provided")

	ErrMissingDestinationAccount          = errors.New("required field destinationAccount must be provided")
	ErrMissingDestinationAccountReference = errors.New("required destinationAccount field reference must be provided")

	ErrMissingRelatedPaymentInitiationReference = fmt.Errorf("required field relatedPaymentInitiation.reference must be provided")

	ErrMissingMetadata = errors.New("required field metadata must be provided")

	ErrMissingCountry = errors.New("required field country must be provided")

	// Metadata Address validation error messages (required when addressLine1 is provided)
	ErrMissingMetadataAddressCity = fmt.Errorf("required metadata field %s must be provided", client.ColumnAddressCityMetadataKey)
	ErrMissingMetadataCountry     = fmt.Errorf("required metadata field %s must be provided", client.ColumnAddressCountryCodeMetadataKey)

	// Metadata Address validation error messages (not required when addressLine1 is not provided)
	ErrMetadataAddressLine2NotRequired   = fmt.Errorf("metadata field %s is not required when addressLine1 is not provided", client.ColumnAddressLine2MetadataKey)
	ErrMetadataAddressCityNotRequired    = fmt.Errorf("metadata field %s is not required when addressLine1 is not provided", client.ColumnAddressCityMetadataKey)
	ErrMetadataAddressStateNotRequired   = fmt.Errorf("metadata field %s is not required when addressLine1 is not provided", client.ColumnAddressStateMetadataKey)
	ErrMetadataAddressCountryNotRequired = fmt.Errorf("metadata field %s is not required when addressLine1 is not provided", client.ColumnAddressCountryCodeMetadataKey)
	ErrMetadataPostalCodeNotRequired     = fmt.Errorf("metadata field %s is not required when addressLine1 is not provided", client.ColumnAddressPostalCodeMetadataKey)

	ErrCountryNotRequired = fmt.Errorf("field country is not required when addressLine1 is not provided")

	// Other metadata validation error messages
	ErrMissingMetadataAllowOverDrafts = fmt.Errorf("required metadata field %s must be provided", client.ColumnAllowOverdraftMetadataKey)
	ErrMissingMetadataHold            = fmt.Errorf("required metadata field %s must be provided", client.ColumnHoldMetadataKey)
	ErrMissingMetadataPayoutType      = fmt.Errorf("required metadata field %s must be provided", client.ColumnPayoutTypeMetadataKey)
	ErrMissingRoutingNumber           = fmt.Errorf("required metadata field %s must be provided", client.ColumnRoutingNumberMetadataKey)
	ErrMissingMetadataReason          = fmt.Errorf("required metadata field %s must be provided", client.ColumnReasonMetadataKey)
	ErrInvalidMetadataPayoutType      = fmt.Errorf("required metadata field %s must be one of: ach, wire, realtime, international-wire", client.ColumnPayoutTypeMetadataKey)
	ErrInvalidMetadataReason          = fmt.Errorf("required metadata field %s must be a valid reason", client.ColumnReasonMetadataKey)
)

type Plugin struct {
    connector.Plugin

    name        string
    connectorID connector.ConnectorID
    logger      logging.Logger

    client                 client.Client
    config                 Config
    supportedWebhooks      map[client.EventCategory]supportedWebhook
    verifier               WebhookVerifier
}

func New(connectorID connector.ConnectorID, name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	client := client.New(ProviderName, config.APIKey, config.Endpoint)
	p := &Plugin{
		Plugin: connector.NewBasePlugin(),

		name:        name,
		connectorID: connectorID,
		logger:      logger,
		client:      client,
		config:      config,
		verifier:    &defaultVerifier{},
	}

	if err := p.initWebhookConfig(); err != nil {
		return p, fmt.Errorf("failed to init webhooks for %s: %w", name, err)
	}
	return p, nil
}

func (p *Plugin) Name() string {
    return p.name
}

func (p *Plugin) Config() connector.PluginInternalConfig {
    return p.config
}

func (p *Plugin) Install(ctx context.Context, req connector.InstallRequest) (connector.InstallResponse, error) {
	return connector.InstallResponse{
		Workflow: workflow(),
	}, nil
}

func (p *Plugin) Uninstall(ctx context.Context, req connector.UninstallRequest) (connector.UninstallResponse, error) {
	if p.client == nil {
		return connector.UninstallResponse{}, connector.ErrNotYetInstalled
	}
	return p.uninstall(ctx, req)
}

func (p *Plugin) FetchNextAccounts(ctx context.Context, req connector.FetchNextAccountsRequest) (connector.FetchNextAccountsResponse, error) {
	if p.client == nil {
		return connector.FetchNextAccountsResponse{}, connector.ErrNotYetInstalled
	}
	return p.fetchNextAccounts(ctx, req)
}

func (p *Plugin) FetchNextBalances(ctx context.Context, req connector.FetchNextBalancesRequest) (connector.FetchNextBalancesResponse, error) {
	if p.client == nil {
		return connector.FetchNextBalancesResponse{}, connector.ErrNotYetInstalled
	}
	return p.fetchNextBalances(ctx, req)
}

func (p *Plugin) FetchNextExternalAccounts(ctx context.Context, req connector.FetchNextExternalAccountsRequest) (connector.FetchNextExternalAccountsResponse, error) {
	if p.client == nil {
		return connector.FetchNextExternalAccountsResponse{}, connector.ErrNotYetInstalled
	}
	return p.fetchNextExternalAccounts(ctx, req)
}

func (p *Plugin) FetchNextPayments(ctx context.Context, req connector.FetchNextPaymentsRequest) (connector.FetchNextPaymentsResponse, error) {
	if p.client == nil {
		return connector.FetchNextPaymentsResponse{}, connector.ErrNotYetInstalled
	}
	return p.fetchNextPayments(ctx, req)
}

func (p *Plugin) CreateBankAccount(ctx context.Context, req connector.CreateBankAccountRequest) (connector.CreateBankAccountResponse, error) {
	if p.client == nil {
		return connector.CreateBankAccountResponse{}, connector.ErrNotYetInstalled
	}
	return p.createBankAccount(ctx, req.BankAccount)
}

func (p *Plugin) CreateTransfer(ctx context.Context, req connector.CreateTransferRequest) (connector.CreateTransferResponse, error) {
	if p.client == nil {
		return connector.CreateTransferResponse{}, connector.ErrNotYetInstalled
	}

	payment, err := p.createTransfer(ctx, req.PaymentInitiation)
	if err != nil {
		return connector.CreateTransferResponse{}, err
	}

	return connector.CreateTransferResponse{
		Payment: payment,
	}, nil
}

func (p *Plugin) CreatePayout(ctx context.Context, req connector.CreatePayoutRequest) (connector.CreatePayoutResponse, error) {
	if p.client == nil {
		return connector.CreatePayoutResponse{}, connector.ErrNotYetInstalled
	}

	return p.createPayout(ctx, req.PaymentInitiation)

}

func (p *Plugin) ReversePayout(ctx context.Context, req connector.ReversePayoutRequest) (connector.ReversePayoutResponse, error) {
	if p.client == nil {
		return connector.ReversePayoutResponse{}, connector.ErrNotYetInstalled
	}

	return p.createReversePayout(ctx, req.PaymentInitiationReversal)

}

func (p *Plugin) CreateWebhooks(ctx context.Context, req connector.CreateWebhooksRequest) (connector.CreateWebhooksResponse, error) {
	if p.client == nil {
		return connector.CreateWebhooksResponse{}, connector.ErrNotYetInstalled
	}
	return p.createWebhooks(ctx, req)
}

func (p *Plugin) VerifyWebhook(ctx context.Context, req connector.VerifyWebhookRequest) (connector.VerifyWebhookResponse, error) {
	if p.client == nil {
		return connector.VerifyWebhookResponse{}, connector.ErrNotYetInstalled
	}
	return p.verifyWebhook(ctx, req)
}

func (p *Plugin) TranslateWebhook(ctx context.Context, req connector.TranslateWebhookRequest) (connector.TranslateWebhookResponse, error) {
	if p.client == nil {
		return connector.TranslateWebhookResponse{}, connector.ErrNotYetInstalled
	}
	return p.translateWebhook(ctx, req)
}

var _ connector.Plugin = &Plugin{}
