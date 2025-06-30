package moov

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/moov/client"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

const ProviderName = "moov"

/*
* Validation error messages
 */
var (
	ErrMissingAmount             = errors.New("required field amount must be provided")
	ErrMissingAsset              = errors.New("required field asset must be provided")
	ErrMissingSourceAccount      = errors.New("required field sourceAccount must be provided")
	ErrMissingDestinationAccount = errors.New("required field destinationAccount must be provided")

	// Source payment method validation errors
	ErrMissingSourcePaymentMethodID            = fmt.Errorf("required field sourcePaymentMethodId must be provided")
	ErrInvalidSourceCardTransactionSource      = fmt.Errorf("source card transaction source must be one of: first-recurring, recurring, unscheduled")
	ErrSourceACHCompanyEntryDescriptionTooLong = fmt.Errorf("source ACH company entry description must be 10 characters or less")
	ErrInvalidSourceACHSecCode                 = fmt.Errorf("source ACH SEC code must be one of: CCD, PPD, TEL, WEB")
	ErrSourceCardDynamicDescriptorTooLong      = fmt.Errorf("source card dynamic descriptor must be 22 characters or less")

	// Destination payment method validation errors
	ErrMissingDestinationPaymentMethodID            = fmt.Errorf("required field destinationPaymentMethodId must be provided")
	ErrDestinationACHCompanyEntryDescriptionTooLong = fmt.Errorf("destination ACH company entry description must be 10 characters or less")
	ErrDestinationACHOriginatingCompanyNameTooLong  = fmt.Errorf("destination ACH originating company name must be 16 characters or less")
	ErrDestinationCardDynamicDescriptorTooLong      = fmt.Errorf("destination card dynamic descriptor must be 22 characters or less")

	// Sales tax validation errors
	ErrMissingSalesTaxValue    = fmt.Errorf("sales tax amount value is required when currency is provided")
	ErrMissingSalesTaxCurrency = fmt.Errorf("sales tax amount currency is required when value is provided")

	// Facilitator fee validation errors
	ErrInvalidFacilitatorFeeTotal             = fmt.Errorf("failed to parse facilitator fee total")
	ErrInvalidFacilitatorFeeMarkup            = fmt.Errorf("failed to parse facilitator fee markup")
	ErrConflictingFacilitatorFeeMarkupFormats = fmt.Errorf("cannot specify both markup and markupDecimal")
	ErrConflictingFacilitatorFeeTotalFormats  = fmt.Errorf("cannot specify both total and totalDecimal")
	ErrConflictingFacilitatorFeeStructures    = fmt.Errorf("cannot specify both total and markup fee structures - use either total/totalDecimal OR markup/markupDecimal")
	ErrMissingFacilitatorFeeStructure         = fmt.Errorf("facilitator fee requires either total/totalDecimal OR markup/markupDecimal")
	ErrInvalidSalesTaxAmount                  = fmt.Errorf("failed to parse sales tax amount")
)

type Plugin struct {
	name   string
	logger logging.Logger

	client client.Client
}

func init() {
	registry.RegisterPlugin(ProviderName, func(connectorID models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{})
}

func New(providerName string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	client, err := client.New(providerName, config.Endpoint, config.PublicKey, config.PrivateKey, config.AccountID)

	if err != nil {
		return nil, err
	}

	return &Plugin{
		name:   providerName,
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
	return p.fetchNextUsers(ctx, req)
}

func (p *Plugin) CreateBankAccount(ctx context.Context, req models.CreateBankAccountRequest) (models.CreateBankAccountResponse, error) {
	return models.CreateBankAccountResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) CreateTransfer(ctx context.Context, req models.CreateTransferRequest) (models.CreateTransferResponse, error) {
	return models.CreateTransferResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) ReverseTransfer(ctx context.Context, req models.ReverseTransferRequest) (models.ReverseTransferResponse, error) {
	return models.ReverseTransferResponse{}, plugins.ErrNotImplemented
}

// Note: Fill only if we cannot have the related payment in the CreateTransfer method
func (p *Plugin) PollTransferStatus(ctx context.Context, req models.PollTransferStatusRequest) (models.PollTransferStatusResponse, error) {
	return models.PollTransferStatusResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) CreatePayout(ctx context.Context, req models.CreatePayoutRequest) (models.CreatePayoutResponse, error) {
	if p.client == nil {
		return models.CreatePayoutResponse{}, plugins.ErrNotYetInstalled
	}

	payout, err := p.createPayout(ctx, req.PaymentInitiation)
	if err != nil {
		return models.CreatePayoutResponse{}, err
	}

	return payout, nil
}

func (p *Plugin) ReversePayout(ctx context.Context, req models.ReversePayoutRequest) (models.ReversePayoutResponse, error) {
	return models.ReversePayoutResponse{}, plugins.ErrNotImplemented
}

// Note: Fill only if we cannot have the related payment in the CreatePayout method
func (p *Plugin) PollPayoutStatus(ctx context.Context, req models.PollPayoutStatusRequest) (models.PollPayoutStatusResponse, error) {
	return models.PollPayoutStatusResponse{}, plugins.ErrNotImplemented
}

// Note: if the connector has webhooks, use this method to create the related
// webhooks on the PSP.
func (p *Plugin) CreateWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	return models.CreateWebhooksResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) VerifyWebhook(ctx context.Context, req models.VerifyWebhookRequest) (models.VerifyWebhookResponse, error) {
	if p.client == nil {
		return models.VerifyWebhookResponse{}, plugins.ErrNotYetInstalled
	}
	return models.VerifyWebhookResponse{}, nil
}

// Note: if the connector has webhooks, use this method to translate incoming
// webhooks to a formance object.
func (p *Plugin) TranslateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	return models.TranslateWebhookResponse{}, plugins.ErrNotImplemented
}

var _ models.Plugin = &Plugin{}
