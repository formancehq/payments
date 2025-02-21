package client

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	gocardless "github.com/gocardless/gocardless-pro-go/v4"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetCustomers(ctx context.Context, pageSize int, after string, before string) ([]GocardlessUser, Cursor, error)
	GetCreditors(ctx context.Context, pageSize int, after string, before string) ([]GocardlessUser, Cursor, error)
	GetExternalAccounts(ctx context.Context, ownerID string, pageSize int, after string, before string) ([]GocardlessGenericAccount, Cursor, error)
	GetPayments(ctx context.Context, payload PaymentPayload, pageSize int, after string, before string) ([]GocardlessPayment, Cursor, error)
	GetMandate(ctx context.Context, mandateId string) (*gocardless.Mandate, error)
	CreateCreditorBankAccount(ctx context.Context, creditor string, ba models.BankAccount) (*gocardless.CreditorBankAccount, error)
	CreateCustomerBankAccount(ctx context.Context, customer string, ba models.BankAccount) (*gocardless.CustomerBankAccount, error)
}

type client struct {
	service *gocardless.Service
}

type Cursor struct {
	After  string `json:"after,omitempty"`
	Before string `json:"before,omitempty"`
}

type GocardlessGenericAccount struct {
	ID                string                 `json:"id,omitempty"`
	CreatedAt         int64                  `json:"created_at,omitempty"`
	AccountHolderName string                 `json:"account_holder_name,omitempty"`
	Currency          string                 `json:"currency,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	AccountType       string                 `json:"account_type,omitempty"`
}

type GocardlessUser struct {
	AddressLine1                        string                                 `json:"address_line1,omitempty"`
	AddressLine2                        string                                 `json:"address_line2,omitempty"`
	AddressLine3                        string                                 `json:"address_line3,omitempty"`
	City                                string                                 `json:"city,omitempty"`
	CountryCode                         string                                 `json:"country_code,omitempty"`
	CreatedAt                           int64                                  `json:"created_at,omitempty"`
	Id                                  string                                 `json:"id,omitempty"`
	Name                                string                                 `json:"name,omitempty"`
	PostalCode                          string                                 `json:"postal_code,omitempty"`
	Region                              string                                 `json:"region,omitempty"`
	CompanyName                         string                                 `json:"company_name,omitempty"`
	DanishIdentityNumber                string                                 `json:"danish_identity_number,omitempty"`
	Email                               string                                 `json:"email,omitempty"`
	Language                            string                                 `json:"language,omitempty"`
	Metadata                            map[string]interface{}                 `json:"metadata,omitempty"`
	PhoneNumber                         string                                 `json:"phone_number,omitempty"`
	SwedishIdentityNumber               string                                 `json:"swedish_identity_number,omitempty"`
	BankReferencePrefix                 string                                 `json:"bank_reference_prefix,omitempty"`
	CanCreateRefunds                    bool                                   `json:"can_create_refunds,omitempty"`
	CreditorType                        string                                 `json:"creditor_type,omitempty"`
	CustomPaymentPagesEnabled           bool                                   `json:"custom_payment_pages_enabled,omitempty"`
	FxPayoutCurrency                    string                                 `json:"fx_payout_currency,omitempty"`
	Links                               *gocardless.CreditorLinks              `json:"links,omitempty"`
	LogoUrl                             string                                 `json:"logo_url,omitempty"`
	MandateImportsEnabled               bool                                   `json:"mandate_imports_enabled,omitempty"`
	MerchantResponsibleForNotifications bool                                   `json:"merchant_responsible_for_notifications,omitempty"`
	SchemeIdentifiers                   []gocardless.CreditorSchemeIdentifiers `json:"scheme_identifiers,omitempty"`
	VerificationStatus                  string                                 `json:"verification_status,omitempty"`
}

const SandboxEndpoint = "https://api-sandbox.gocardless.com"

func New(connectorName string, endpoint string, accessToken string) (*client, error) {
	if endpoint == "" {
		endpoint = SandboxEndpoint
	}

	config, err := gocardless.NewConfig(accessToken, gocardless.WithEndpoint(endpoint))

	if err != nil {
		return nil, err
	}

	service, err := gocardless.New(config)

	if err != nil {
		return nil, err
	}

	client := &client{
		service: service,
	}

	return client, nil
}
