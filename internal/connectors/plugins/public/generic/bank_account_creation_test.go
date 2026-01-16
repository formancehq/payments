package generic

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/generic/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestCreateBankAccount_Plugin_NotInstalled tests that CreateBankAccount returns
// ErrNotYetInstalled when the client is not initialized (plugin not installed)
func TestCreateBankAccount_Plugin_NotInstalled(t *testing.T) {
	t.Parallel()

	plugin := &Plugin{client: nil}

	accountNumber := "123456789"
	ba := models.BankAccount{
		Name:          "Test Bank Account",
		AccountNumber: &accountNumber,
	}

	_, err := plugin.CreateBankAccount(context.Background(), models.CreateBankAccountRequest{BankAccount: ba})
	require.Error(t, err)
	require.ErrorIs(t, err, plugins.ErrNotYetInstalled)
}

// TestCreateBankAccount_Plugin_ForwardFlow tests the full forward flow at the plugin level
// This simulates what happens when BankAccountsForwardToConnector API is called
func TestCreateBankAccount_Plugin_ForwardFlow(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	now := time.Now().UTC()
	accountNumber := "123456789"
	iban := "DE89370400440532013000"
	swiftCode := "COBADEFFXXX"
	country := "DE"

	// Simulate the bank account data that would come from the forward request
	ba := models.BankAccount{
		Name:          "Test Bank Account",
		AccountNumber: &accountNumber,
		IBAN:          &iban,
		SwiftBicCode:  &swiftCode,
		Country:       &country,
		Metadata: map[string]string{
			"forwarded": "true",
		},
	}

	mockClient.EXPECT().CreateBankAccount(gomock.Any(), &client.BankAccountRequest{
		Name:          "Test Bank Account",
		AccountNumber: &accountNumber,
		IBAN:          &iban,
		SwiftBicCode:  &swiftCode,
		Country:       &country,
		Metadata:      map[string]string{"forwarded": "true"},
	}).Return(&client.BankAccountResponse{
		Id:            "external_bank_account_123",
		Name:          "Test Bank Account",
		AccountNumber: &accountNumber,
		IBAN:          &iban,
		SwiftBicCode:  &swiftCode,
		Country:       &country,
		CreatedAt:     now.Format(time.RFC3339),
		Metadata:      map[string]string{"forwarded": "true"},
	}, nil)

	// Call the plugin's CreateBankAccount method (this is what the activity calls)
	resp, err := plugin.CreateBankAccount(context.Background(), models.CreateBankAccountRequest{BankAccount: ba})
	require.NoError(t, err)
	require.Equal(t, "external_bank_account_123", resp.RelatedAccount.Reference)
	require.Equal(t, "Test Bank Account", *resp.RelatedAccount.Name)
	require.NotNil(t, resp.RelatedAccount.Raw)

	// Verify metadata is populated correctly
	require.Equal(t, accountNumber, resp.RelatedAccount.Metadata[models.AccountAccountNumberMetadataKey])
	require.Equal(t, iban, resp.RelatedAccount.Metadata[models.AccountIBANMetadataKey])
	require.Equal(t, swiftCode, resp.RelatedAccount.Metadata[models.AccountSwiftBicCodeMetadataKey])
	require.Equal(t, country, resp.RelatedAccount.Metadata[models.AccountBankAccountCountryMetadataKey])
}

func TestCreateBankAccount_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	now := time.Now().UTC()
	accountNumber := "123456789"
	iban := "DE89370400440532013000"
	swiftCode := "COBADEFFXXX"
	country := "DE"

	ba := models.BankAccount{
		Name:          "Test Bank Account",
		AccountNumber: &accountNumber,
		IBAN:          &iban,
		SwiftBicCode:  &swiftCode,
		Country:       &country,
		Metadata: map[string]string{
			"test_key": "test_value",
		},
	}

	mockClient.EXPECT().CreateBankAccount(gomock.Any(), gomock.Any()).Return(&client.BankAccountResponse{
		Id:            "bank_account_123",
		Name:          "Test Bank Account",
		AccountNumber: &accountNumber,
		IBAN:          &iban,
		SwiftBicCode:  &swiftCode,
		Country:       &country,
		CreatedAt:     now.Format(time.RFC3339),
		Metadata:      map[string]string{"test_key": "test_value"},
	}, nil)

	resp, err := plugin.CreateBankAccount(context.Background(), models.CreateBankAccountRequest{BankAccount: ba})
	require.NoError(t, err)
	require.Equal(t, "bank_account_123", resp.RelatedAccount.Reference)
	require.Equal(t, "Test Bank Account", *resp.RelatedAccount.Name)
	require.NotNil(t, resp.RelatedAccount.Raw)
}

func TestCreateBankAccount_WithAccountNumberOnly(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	now := time.Now().UTC()
	accountNumber := "123456789"

	ba := models.BankAccount{
		Name:          "Test Bank Account",
		AccountNumber: &accountNumber,
	}

	mockClient.EXPECT().CreateBankAccount(gomock.Any(), gomock.Any()).Return(&client.BankAccountResponse{
		Id:            "bank_account_123",
		Name:          "Test Bank Account",
		AccountNumber: &accountNumber,
		CreatedAt:     now.Format(time.RFC3339),
	}, nil)

	resp, err := plugin.CreateBankAccount(context.Background(), models.CreateBankAccountRequest{BankAccount: ba})
	require.NoError(t, err)
	require.Equal(t, "bank_account_123", resp.RelatedAccount.Reference)
}

func TestCreateBankAccount_WithIBANOnly(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	now := time.Now().UTC()
	iban := "DE89370400440532013000"

	ba := models.BankAccount{
		Name: "Test Bank Account",
		IBAN: &iban,
	}

	mockClient.EXPECT().CreateBankAccount(gomock.Any(), gomock.Any()).Return(&client.BankAccountResponse{
		Id:        "bank_account_123",
		Name:      "Test Bank Account",
		IBAN:      &iban,
		CreatedAt: now.Format(time.RFC3339),
	}, nil)

	resp, err := plugin.CreateBankAccount(context.Background(), models.CreateBankAccountRequest{BankAccount: ba})
	require.NoError(t, err)
	require.Equal(t, "bank_account_123", resp.RelatedAccount.Reference)
}

func TestCreateBankAccount_ClientError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	accountNumber := "123456789"

	ba := models.BankAccount{
		Name:          "Test Bank Account",
		AccountNumber: &accountNumber,
	}

	mockClient.EXPECT().CreateBankAccount(gomock.Any(), gomock.Any()).Return(nil, errors.New("network error"))

	resp, err := plugin.CreateBankAccount(context.Background(), models.CreateBankAccountRequest{BankAccount: ba})
	require.Error(t, err)
	require.Contains(t, err.Error(), "network error")
	require.Equal(t, models.CreateBankAccountResponse{}, resp)
}

func TestValidateBankAccountRequest(t *testing.T) {
	t.Parallel()

	plugin := &Plugin{}

	t.Run("valid with account number", func(t *testing.T) {
		accountNumber := "123456789"
		ba := models.BankAccount{
			Name:          "Test Account",
			AccountNumber: &accountNumber,
		}

		err := plugin.validateBankAccountRequest(ba)
		require.NoError(t, err)
	})

	t.Run("valid with IBAN", func(t *testing.T) {
		iban := "DE89370400440532013000"
		ba := models.BankAccount{
			Name: "Test Account",
			IBAN: &iban,
		}

		err := plugin.validateBankAccountRequest(ba)
		require.NoError(t, err)
	})

	t.Run("valid with both account number and IBAN", func(t *testing.T) {
		accountNumber := "123456789"
		iban := "DE89370400440532013000"
		ba := models.BankAccount{
			Name:          "Test Account",
			AccountNumber: &accountNumber,
			IBAN:          &iban,
		}

		err := plugin.validateBankAccountRequest(ba)
		require.NoError(t, err)
	})

	t.Run("missing name", func(t *testing.T) {
		accountNumber := "123456789"
		ba := models.BankAccount{
			Name:          "",
			AccountNumber: &accountNumber,
		}

		err := plugin.validateBankAccountRequest(ba)
		require.Error(t, err)
		require.Contains(t, err.Error(), "name is required")
	})

	t.Run("missing both account number and IBAN", func(t *testing.T) {
		ba := models.BankAccount{
			Name: "Test Account",
		}

		err := plugin.validateBankAccountRequest(ba)
		require.Error(t, err)
		require.Contains(t, err.Error(), "either account number or IBAN is required")
	})

	t.Run("empty account number and nil IBAN", func(t *testing.T) {
		emptyAccountNumber := ""
		ba := models.BankAccount{
			Name:          "Test Account",
			AccountNumber: &emptyAccountNumber,
		}

		err := plugin.validateBankAccountRequest(ba)
		require.Error(t, err)
		require.Contains(t, err.Error(), "either account number or IBAN is required")
	})

	t.Run("nil account number and empty IBAN", func(t *testing.T) {
		emptyIBAN := ""
		ba := models.BankAccount{
			Name: "Test Account",
			IBAN: &emptyIBAN,
		}

		err := plugin.validateBankAccountRequest(ba)
		require.Error(t, err)
		require.Contains(t, err.Error(), "either account number or IBAN is required")
	})
}

func TestBankAccountResponseToAccount(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	t.Run("full response with all fields", func(t *testing.T) {
		accountNumber := "123456789"
		iban := "DE89370400440532013000"
		swiftCode := "COBADEFFXXX"
		country := "DE"

		resp := &client.BankAccountResponse{
			Id:            "bank_account_123",
			Name:          "Test Bank Account",
			AccountNumber: &accountNumber,
			IBAN:          &iban,
			SwiftBicCode:  &swiftCode,
			Country:       &country,
			CreatedAt:     now.Format(time.RFC3339),
			Metadata:      map[string]string{"key": "value"},
		}

		result, err := bankAccountResponseToAccount(resp)
		require.NoError(t, err)
		require.Equal(t, "bank_account_123", result.RelatedAccount.Reference)
		require.Equal(t, "Test Bank Account", *result.RelatedAccount.Name)
		require.Equal(t, accountNumber, result.RelatedAccount.Metadata[models.AccountAccountNumberMetadataKey])
		require.Equal(t, iban, result.RelatedAccount.Metadata[models.AccountIBANMetadataKey])
		require.Equal(t, swiftCode, result.RelatedAccount.Metadata[models.AccountSwiftBicCodeMetadataKey])
		require.Equal(t, country, result.RelatedAccount.Metadata[models.AccountBankAccountCountryMetadataKey])
		require.NotNil(t, result.RelatedAccount.Raw)
	})

	t.Run("response with nil metadata creates new map", func(t *testing.T) {
		accountNumber := "123456789"

		resp := &client.BankAccountResponse{
			Id:            "bank_account_123",
			Name:          "Test Bank Account",
			AccountNumber: &accountNumber,
			CreatedAt:     now.Format(time.RFC3339),
			Metadata:      nil,
		}

		result, err := bankAccountResponseToAccount(resp)
		require.NoError(t, err)
		require.NotNil(t, result.RelatedAccount.Metadata)
		require.Equal(t, accountNumber, result.RelatedAccount.Metadata[models.AccountAccountNumberMetadataKey])
	})

	t.Run("response with partial fields", func(t *testing.T) {
		resp := &client.BankAccountResponse{
			Id:        "bank_account_123",
			Name:      "Test Bank Account",
			CreatedAt: now.Format(time.RFC3339),
		}

		result, err := bankAccountResponseToAccount(resp)
		require.NoError(t, err)
		require.Equal(t, "bank_account_123", result.RelatedAccount.Reference)
		// Should not have metadata keys for nil fields
		_, hasAccountNumber := result.RelatedAccount.Metadata[models.AccountAccountNumberMetadataKey]
		require.False(t, hasAccountNumber)
	})

	t.Run("invalid created at date", func(t *testing.T) {
		resp := &client.BankAccountResponse{
			Id:        "bank_account_123",
			Name:      "Test Bank Account",
			CreatedAt: "invalid-date",
		}

		_, err := bankAccountResponseToAccount(resp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse created at")
	})
}
