package client

import (
	"context"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/models"
	gocardless "github.com/gocardless/gocardless-pro-go/v4"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetCustomers(ctx context.Context, pageSize int, after string) ([]GocardlessUser, Cursor, error)
	GetCreditors(ctx context.Context, pageSize int, after string) ([]GocardlessUser, Cursor, error)
	GetExternalAccounts(ctx context.Context, ownerID string, pageSize int, after string) ([]GocardlessGenericAccount, Cursor, error)
	GetPayments(ctx context.Context, pageSize int, after string) ([]GocardlessPayment, Cursor, error)
	GetMandate(ctx context.Context, mandateId string) (*gocardless.Mandate, error)
	CreateCreditorBankAccount(ctx context.Context, creditor string, ba models.BankAccount) (GocardlessGenericAccount, error)
	CreateCustomerBankAccount(ctx context.Context, customer string, ba models.BankAccount) (GocardlessGenericAccount, error)
	NewWithService(service GoCardlessService)
}

type GoCardlessService interface {
	GetGocardlessCustomers(ctx context.Context, params gocardless.CustomerListParams, opts ...gocardless.RequestOption) (*gocardless.CustomerListResult, error)
	GetGocardlessCreditors(ctx context.Context, params gocardless.CreditorListParams, opts ...gocardless.RequestOption) (*gocardless.CreditorListResult, error)
	GetMandate(ctx context.Context, identity string, opts ...gocardless.RequestOption) (*gocardless.Mandate, error)
	GetGocardlessPayments(ctx context.Context, p gocardless.PaymentListParams, opts ...gocardless.RequestOption) (*gocardless.PaymentListResult, error)
	CreateGocardlessCreditorBankAccount(ctx context.Context, params gocardless.CreditorBankAccountCreateParams, opts ...gocardless.RequestOption) (*gocardless.CreditorBankAccount, error)
	CreateGocardlessCustomerBankAccount(ctx context.Context, params gocardless.CustomerBankAccountCreateParams, opts ...gocardless.RequestOption) (*gocardless.CustomerBankAccount, error)
	GetGocardlessCustomerBankAccounts(ctx context.Context, params gocardless.CustomerBankAccountListParams, opts ...gocardless.RequestOption) (*gocardless.CustomerBankAccountListResult, error)
	GetGocardlessCreditorBankAccounts(ctx context.Context, params gocardless.CreditorBankAccountListParams, opts ...gocardless.RequestOption) (*gocardless.CreditorBankAccountListResult, error)
	GetGocardlessPayout(ctx context.Context, identity string, opts ...gocardless.RequestOption) (*gocardless.Payout, error)
}

type client struct {
	service GoCardlessService

	endpoint           string
	accessToken        string
	shouldFetchMandate bool
}

type Cursor struct {
	After string `json:"after,omitempty"`
}

type GocardlessGenericAccount struct {
	ID                string                 `json:"id,omitempty"`
	CreatedAt         time.Time              `json:"created_at,omitempty"`
	AccountHolderName string                 `json:"account_holder_name,omitempty"`
	Currency          string                 `json:"currency,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	AccountType       string                 `json:"account_type,omitempty"`
}

type GocardlessUser struct {
	AddressLine1                        string                    `json:"address_line1,omitempty"`
	AddressLine2                        string                    `json:"address_line2,omitempty"`
	AddressLine3                        string                    `json:"address_line3,omitempty"`
	City                                string                    `json:"city,omitempty"`
	CountryCode                         string                    `json:"country_code,omitempty"`
	CreatedAt                           time.Time                 `json:"created_at,omitempty"`
	Id                                  string                    `json:"id,omitempty"`
	Name                                string                    `json:"name,omitempty"`
	PostalCode                          string                    `json:"postal_code,omitempty"`
	Region                              string                    `json:"region,omitempty"`
	CompanyName                         string                    `json:"company_name,omitempty"`
	DanishIdentityNumber                string                    `json:"danish_identity_number,omitempty"`
	Email                               string                    `json:"email,omitempty"`
	Language                            string                    `json:"language,omitempty"`
	Metadata                            map[string]interface{}    `json:"metadata,omitempty"`
	PhoneNumber                         string                    `json:"phone_number,omitempty"`
	SwedishIdentityNumber               string                    `json:"swedish_identity_number,omitempty"`
	BankReferencePrefix                 string                    `json:"bank_reference_prefix,omitempty"`
	CanCreateRefunds                    bool                      `json:"can_create_refunds,omitempty"`
	CreditorType                        string                    `json:"creditor_type,omitempty"`
	CustomPaymentPagesEnabled           bool                      `json:"custom_payment_pages_enabled,omitempty"`
	FxPayoutCurrency                    string                    `json:"fx_payout_currency,omitempty"`
	Links                               *gocardless.CreditorLinks `json:"links,omitempty"`
	LogoUrl                             string                    `json:"logo_url,omitempty"`
	MandateImportsEnabled               bool                      `json:"mandate_imports_enabled,omitempty"`
	MerchantResponsibleForNotifications bool                      `json:"merchant_responsible_for_notifications,omitempty"`
	VerificationStatus                  string                    `json:"verification_status,omitempty"`
}

const SandboxEndpoint = "https://api-sandbox.gocardless.com"

type serviceWrapper struct {
	*gocardless.Service
}

func (s *serviceWrapper) CreateGocardlessCreditorBankAccount(ctx context.Context, params gocardless.CreditorBankAccountCreateParams, opts ...gocardless.RequestOption) (*gocardless.CreditorBankAccount, error) {
	return s.Service.CreditorBankAccounts.Create(ctx, params, opts...)
}

func (s *serviceWrapper) CreateGocardlessCustomerBankAccount(ctx context.Context, params gocardless.CustomerBankAccountCreateParams, opts ...gocardless.RequestOption) (*gocardless.CustomerBankAccount, error) {
	return s.Service.CustomerBankAccounts.Create(ctx, params, opts...)
}

func (s *serviceWrapper) GetGocardlessCustomers(ctx context.Context, params gocardless.CustomerListParams, opts ...gocardless.RequestOption) (*gocardless.CustomerListResult, error) {
	return s.Service.Customers.List(ctx, params, opts...)
}

func (s *serviceWrapper) GetGocardlessCreditors(ctx context.Context, params gocardless.CreditorListParams, opts ...gocardless.RequestOption) (*gocardless.CreditorListResult, error) {
	return s.Service.Creditors.List(ctx, params, opts...)
}

func (s *serviceWrapper) GetGocardlessCustomerBankAccounts(ctx context.Context, params gocardless.CustomerBankAccountListParams, opts ...gocardless.RequestOption) (*gocardless.CustomerBankAccountListResult, error) {
	return s.Service.CustomerBankAccounts.List(ctx, params, opts...)
}

func (s *serviceWrapper) GetGocardlessCreditorBankAccounts(ctx context.Context, params gocardless.CreditorBankAccountListParams, opts ...gocardless.RequestOption) (*gocardless.CreditorBankAccountListResult, error) {
	return s.Service.CreditorBankAccounts.List(ctx, params, opts...)
}

func (s *serviceWrapper) GetMandate(ctx context.Context, identity string, opts ...gocardless.RequestOption) (*gocardless.Mandate, error) {
	return s.Service.Mandates.Get(ctx, identity, opts...)
}

func (s *serviceWrapper) GetGocardlessPayments(ctx context.Context, p gocardless.PaymentListParams, opts ...gocardless.RequestOption) (*gocardless.PaymentListResult, error) {
	return s.Service.Payments.List(ctx, p, opts...)
}

func (s *serviceWrapper) GetGocardlessPayout(ctx context.Context, identity string, opts ...gocardless.RequestOption) (*gocardless.Payout, error) {
	return s.Service.Payouts.Get(ctx, identity, opts...)
}

func New(connectorName string, endpoint string, accessToken string, shouldFetchMandate bool) (*client, error) {
	if endpoint == "" {
		endpoint = SandboxEndpoint
	}

	config, err := gocardless.NewConfig(accessToken,
		gocardless.WithEndpoint(endpoint),
		gocardless.WithClient(metrics.NewHTTPClient(connectorName, models.DefaultConnectorClientTimeout)),
	)

	if err != nil {
		return nil, err
	}

	service, err := gocardless.New(config)

	if err != nil {
		return nil, err
	}

	client := &client{
		service: &serviceWrapper{service},

		endpoint:           endpoint,
		accessToken:        accessToken,
		shouldFetchMandate: shouldFetchMandate,
	}

	return client, nil
}

func (c *client) NewWithService(service GoCardlessService) {
	c.service = service
}
