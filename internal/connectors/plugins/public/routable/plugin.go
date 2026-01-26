package routable

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/routable/client"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

const ProviderName = "routable"

func init() {
	registry.RegisterPlugin(ProviderName, models.PluginTypePSP, func(_ models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{})
}

type Plugin struct {
	models.Plugin

	name   string
	logger logging.Logger

	client              client.Client
	webhookSharedSecret string
	actingTeamMemberID  string
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	cli := client.New(ProviderName, config.APIToken, config.Endpoint)

	return &Plugin{
		Plugin: plugins.NewBasePlugin(),

		name:                name,
		logger:              logger,
		client:              cli,
		webhookSharedSecret: config.WebhookSharedSecret,
		actingTeamMemberID:  config.ActingTeamMemberID,
	}, nil
}

func (p *Plugin) Name() string {
	return p.name
}

func (p *Plugin) Install(_ context.Context, req models.InstallRequest) (models.InstallResponse, error) {
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
	return p.fetchNextExternalAccounts(ctx, req)
}

func (p *Plugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	if p.client == nil {
		return models.FetchNextPaymentsResponse{}, plugins.ErrNotYetInstalled
	}
	return p.fetchNextPayments(ctx, req)
}

func (p *Plugin) FetchNextOthers(ctx context.Context, req models.FetchNextOthersRequest) (models.FetchNextOthersResponse, error) {
	return models.FetchNextOthersResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) CreateBankAccount(ctx context.Context, req models.CreateBankAccountRequest) (models.CreateBankAccountResponse, error) {
	if p.client == nil {
		return models.CreateBankAccountResponse{}, plugins.ErrNotYetInstalled
	}

	ba := req.BankAccount
	md := ba.Metadata

	// Company
	displayName := ba.Name
	companyType := "business"
	if v := strings.ToLower(md["spec.formance.com/psu.type"]); v == "personal" {
		companyType = "personal"
	}
	var namePtr, bizPtr *string
	if companyType == "business" {
		bizPtr = &displayName
	} else {
		namePtr = &displayName
	}
	comp, err := p.client.CreateCompany(ctx, &client.CreateCompanyRequest{
		Type:             companyType,
		Name:             namePtr,
		BusinessName:     bizPtr,
		ActingTeamMember: p.actingTeamMemberID,
		IsVendor:         true,
		IsCustomer:       false,
	})
	if err != nil {
		return models.CreateBankAccountResponse{}, err
	}

	// Contact (actionable)
	email := md[models.BankAccountOwnerEmailMetadataKey]
	if email != "" {
		_, _ = p.client.CreateContact(ctx, comp.ID, &client.CreateContactRequest{
			FirstName:          "",
			LastName:           "",
			Email:              email,
			DefaultForPayables: "actionable",
			ActingTeamMember:   p.actingTeamMemberID,
		})
	}

	// Payment method (bank)
	var acctNum, routing, iban, country, currency string
	// Prefer BankAccount fields first
	if ba.AccountNumber != nil {
		acctNum = *ba.AccountNumber
	}
	if ba.IBAN != nil {
		iban = *ba.IBAN
	}
	if ba.SwiftBicCode != nil {
		routing = *ba.SwiftBicCode
	}
	if ba.Country != nil {
		country = *ba.Country
	}
	// Fallback to metadata for any missing
	if acctNum == "" {
		acctNum = md[models.AccountAccountNumberMetadataKey]
	}
	if iban == "" {
		iban = md[models.AccountIBANMetadataKey]
	}
	// Try dedicated routing if provided (non-standard key), else fallback to swiftBicCode from metadata
	if v := md["formance.spec/owner/routingNumber"]; v != "" {
		routing = v
	} else if routing == "" {
		routing = md[models.AccountSwiftBicCodeMetadataKey]
	}
	if country == "" {
		country = md[models.AccountBankAccountCountryMetadataKey]
	}
	// Heuristic currency default
	if currency == "" && strings.EqualFold(country, "US") {
		currency = "USD"
	}

	// Create payment method only if we have sufficient details
	if iban != "" || (acctNum != "" && routing != "") {
		_, err = p.client.CreateBankPaymentMethod(ctx, comp.ID, &client.CreateBankPaymentMethodRequest{
			Type: "bank",
			TypeDetails: client.CreateBankPaymentMethodDetails{
				AccountType:   "checking",
				AccountNumber: acctNum,
				RoutingNumber: routing,
				Iban:          iban,
			},
		})
		if err != nil {
			return models.CreateBankAccountResponse{}, err
		}
	}

	raw, _ := json.Marshal(map[string]any{"company_id": comp.ID})
	psp := models.PSPAccount{
		Reference: comp.ID,
		CreatedAt: time.Now().UTC(),
		Name:      &displayName,
		Metadata:  map[string]string{"spec.formance.com/generic_provider": ProviderName},
		Raw:       raw,
	}
	return models.CreateBankAccountResponse{RelatedAccount: psp}, nil
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

	payment, err := p.createPayout(ctx, req.PaymentInitiation)
	if err != nil {
		return models.CreatePayoutResponse{}, err
	}

	return models.CreatePayoutResponse{
		Payment: payment,
	}, nil
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

// Note: if the connector has webhooks, use this method to translate incoming
// webhooks to a formance object.
func (p *Plugin) TranslateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	return models.TranslateWebhookResponse{}, plugins.ErrNotImplemented
}

var _ models.Plugin = &Plugin{}
