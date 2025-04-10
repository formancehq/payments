package column

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/column/client"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

const ProviderName = "column"

func init() {
	registry.RegisterPlugin(ProviderName, func(name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{})
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
	name   string
	logger logging.Logger

	client         client.Client
	webhookConfigs map[client.EventCategory]webhookConfig
	verifier       WebhookVerifier
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	client := client.New(ProviderName, config.APIKey, config.Endpoint)
	p := &Plugin{
		name:     name,
		logger:   logger,
		client:   client,
		verifier: &defaultVerifier{},
	}

	p.initWebhookConfig()

	return p, nil
}

func (p *Plugin) Name() string {
	return p.name
}

func (p *Plugin) Install(ctx context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	var isSecretMissing bool

	configs := make([]models.PSPWebhookConfig, 0, len(p.webhookConfigs))
	for name, config := range p.webhookConfigs {
		if config.secret == "" {
			isSecretMissing = true
		}
		configs = append(configs, models.PSPWebhookConfig{
			Name:    string(name),
			URLPath: config.urlPath,
		})
	}

	if isSecretMissing {
		webhooks, err := p.client.ListEventSubscriptions(ctx)
		if err != nil {
			return models.InstallResponse{}, err
		}
		for _, webhook := range webhooks {
			if !strings.Contains(webhook.URL, req.ConnectorID) {
				continue
			}
			eventCategory := client.EventCategory(webhook.EnabledEvents[0])
			config := p.webhookConfigs[eventCategory]
			config.secret = webhook.Secret
			p.webhookConfigs[eventCategory] = config
		}
	}
	return models.InstallResponse{
		Workflow:        workflow(),
		WebhooksConfigs: configs,
	}, nil
}

func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	if p.client == nil {
		return models.UninstallResponse{}, plugins.ErrNotYetInstalled
	}
	return p.uninstall(ctx, req)
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

func (p *Plugin) FetchNextOthers(ctx context.Context, req models.FetchNextOthersRequest) (models.FetchNextOthersResponse, error) {
	if p.client == nil {
		return models.FetchNextOthersResponse{}, plugins.ErrNotYetInstalled
	}
	return models.FetchNextOthersResponse{}, plugins.ErrNotImplemented
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

	payment, err := p.createTransfer(ctx, req.PaymentInitiation)
	if err != nil {
		return models.CreateTransferResponse{}, err
	}

	return models.CreateTransferResponse{
		Payment: payment,
	}, nil
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

	return p.createPayout(ctx, req.PaymentInitiation)

}

func (p *Plugin) ReversePayout(ctx context.Context, req models.ReversePayoutRequest) (models.ReversePayoutResponse, error) {
	if p.client == nil {
		return models.ReversePayoutResponse{}, plugins.ErrNotYetInstalled
	}

	return p.createReversePayout(ctx, req.PaymentInitiationReversal)

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
	return p.createWebhooks(ctx, req)
}

func (p *Plugin) TranslateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	if p.client == nil {
		return models.TranslateWebhookResponse{}, plugins.ErrNotYetInstalled
	}
	return p.translateWebhook(ctx, req)
}

var _ models.Plugin = &Plugin{}
