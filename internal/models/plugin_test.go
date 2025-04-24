package models_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestMockPlugin(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPlugin := models.NewMockPlugin(ctrl)

	t.Run("Name", func(t *testing.T) {
		t.Parallel()
		// Given
		
		mockPlugin.EXPECT().Name().Return("test-plugin")
		
		name := mockPlugin.Name()
		
		assert.Equal(t, "test-plugin", name)
	})

	t.Run("Install", func(t *testing.T) {
		t.Parallel()
		// Given
		
		ctx := context.Background()
		req := models.InstallRequest{
			ConnectorID: "test-connector",
		}
		resp := models.InstallResponse{
			Workflow:        models.ConnectorTasksTree{},
			WebhooksConfigs: []models.PSPWebhookConfig{},
		}
		
		mockPlugin.EXPECT().Install(ctx, req).Return(resp, nil)
		
		result, err := mockPlugin.Install(ctx, req)
		// When/Then
		assert.NoError(t, err)
		assert.Equal(t, resp.Workflow, result.Workflow)
		assert.Equal(t, resp.WebhooksConfigs, result.WebhooksConfigs)
	})

	t.Run("Uninstall", func(t *testing.T) {
		t.Parallel()
		// Given
		
		ctx := context.Background()
		req := models.UninstallRequest{
			ConnectorID: "test-connector",
		}
		resp := models.UninstallResponse{}
		
		mockPlugin.EXPECT().Uninstall(ctx, req).Return(resp, nil)
		
		result, err := mockPlugin.Uninstall(ctx, req)
		// When/Then
		assert.NoError(t, err)
		assert.Equal(t, resp, result)
	})

	t.Run("CreateBankAccount", func(t *testing.T) {
		t.Parallel()
		// Given
		
		ctx := context.Background()
		req := models.CreateBankAccountRequest{
			BankAccount: models.BankAccount{},
		}
		resp := models.CreateBankAccountResponse{
			RelatedAccount: models.PSPAccount{},
		}
		
		mockPlugin.EXPECT().CreateBankAccount(ctx, req).Return(resp, nil)
		
		result, err := mockPlugin.CreateBankAccount(ctx, req)
		// When/Then
		assert.NoError(t, err)
		assert.Equal(t, resp, result)
	})

	t.Run("CreatePayout", func(t *testing.T) {
		t.Parallel()
		// Given
		
		ctx := context.Background()
		req := models.CreatePayoutRequest{
			PaymentInitiation: models.PSPPaymentInitiation{},
		}
		resp := models.CreatePayoutResponse{
			Payment:         nil,
			PollingPayoutID: nil,
		}
		
		mockPlugin.EXPECT().CreatePayout(ctx, req).Return(resp, nil)
		
		result, err := mockPlugin.CreatePayout(ctx, req)
		// When/Then
		assert.NoError(t, err)
		assert.Equal(t, resp, result)
	})

	t.Run("CreateTransfer", func(t *testing.T) {
		t.Parallel()
		// Given
		
		ctx := context.Background()
		req := models.CreateTransferRequest{
			PaymentInitiation: models.PSPPaymentInitiation{},
		}
		resp := models.CreateTransferResponse{
			Payment:           nil,
			PollingTransferID: nil,
		}
		
		mockPlugin.EXPECT().CreateTransfer(ctx, req).Return(resp, nil)
		
		result, err := mockPlugin.CreateTransfer(ctx, req)
		// When/Then
		assert.NoError(t, err)
		assert.Equal(t, resp, result)
	})

	t.Run("CreateWebhooks", func(t *testing.T) {
		t.Parallel()
		// Given
		
		ctx := context.Background()
		req := models.CreateWebhooksRequest{
			ConnectorID:    "test-connector",
			WebhookBaseUrl: "https://example.com/webhooks",
			FromPayload:    json.RawMessage(`{}`),
		}
		resp := models.CreateWebhooksResponse{
			Others: []models.PSPOther{},
		}
		
		mockPlugin.EXPECT().CreateWebhooks(ctx, req).Return(resp, nil)
		
		result, err := mockPlugin.CreateWebhooks(ctx, req)
		// When/Then
		assert.NoError(t, err)
		assert.Equal(t, resp, result)
	})

	t.Run("FetchNextAccounts", func(t *testing.T) {
		t.Parallel()
		// Given
		
		ctx := context.Background()
		req := models.FetchNextAccountsRequest{
			FromPayload: json.RawMessage(`{}`),
			State:       json.RawMessage(`{}`),
			PageSize:    10,
		}
		resp := models.FetchNextAccountsResponse{
			Accounts: []models.PSPAccount{},
			NewState: json.RawMessage(`{}`),
			HasMore:  false,
		}
		
		mockPlugin.EXPECT().FetchNextAccounts(ctx, req).Return(resp, nil)
		
		result, err := mockPlugin.FetchNextAccounts(ctx, req)
		// When/Then
		assert.NoError(t, err)
		assert.Equal(t, resp, result)
	})

	t.Run("FetchNextBalances", func(t *testing.T) {
		t.Parallel()
		// Given
		
		ctx := context.Background()
		req := models.FetchNextBalancesRequest{
			FromPayload: json.RawMessage(`{}`),
			State:       json.RawMessage(`{}`),
			PageSize:    10,
		}
		resp := models.FetchNextBalancesResponse{
			Balances: []models.PSPBalance{},
			NewState: json.RawMessage(`{}`),
			HasMore:  false,
		}
		
		mockPlugin.EXPECT().FetchNextBalances(ctx, req).Return(resp, nil)
		
		result, err := mockPlugin.FetchNextBalances(ctx, req)
		// When/Then
		assert.NoError(t, err)
		assert.Equal(t, resp, result)
	})

	t.Run("FetchNextExternalAccounts", func(t *testing.T) {
		t.Parallel()
		// Given
		
		ctx := context.Background()
		req := models.FetchNextExternalAccountsRequest{
			FromPayload: json.RawMessage(`{}`),
			State:       json.RawMessage(`{}`),
			PageSize:    10,
		}
		resp := models.FetchNextExternalAccountsResponse{
			ExternalAccounts: []models.PSPAccount{},
			NewState:         json.RawMessage(`{}`),
			HasMore:          false,
		}
		
		mockPlugin.EXPECT().FetchNextExternalAccounts(ctx, req).Return(resp, nil)
		
		result, err := mockPlugin.FetchNextExternalAccounts(ctx, req)
		// When/Then
		assert.NoError(t, err)
		assert.Equal(t, resp, result)
	})

	t.Run("FetchNextOthers", func(t *testing.T) {
		t.Parallel()
		// Given
		
		ctx := context.Background()
		req := models.FetchNextOthersRequest{
			Name:        "test",
			FromPayload: json.RawMessage(`{}`),
			State:       json.RawMessage(`{}`),
			PageSize:    10,
		}
		resp := models.FetchNextOthersResponse{
			Others:   []models.PSPOther{},
			NewState: json.RawMessage(`{}`),
			HasMore:  false,
		}
		
		mockPlugin.EXPECT().FetchNextOthers(ctx, req).Return(resp, nil)
		
		result, err := mockPlugin.FetchNextOthers(ctx, req)
		// When/Then
		assert.NoError(t, err)
		assert.Equal(t, resp, result)
	})

	t.Run("FetchNextPayments", func(t *testing.T) {
		t.Parallel()
		// Given
		
		ctx := context.Background()
		req := models.FetchNextPaymentsRequest{
			FromPayload: json.RawMessage(`{}`),
			State:       json.RawMessage(`{}`),
			PageSize:    10,
		}
		resp := models.FetchNextPaymentsResponse{
			Payments: []models.PSPPayment{},
			NewState: json.RawMessage(`{}`),
			HasMore:  false,
		}
		
		mockPlugin.EXPECT().FetchNextPayments(ctx, req).Return(resp, nil)
		
		result, err := mockPlugin.FetchNextPayments(ctx, req)
		// When/Then
		assert.NoError(t, err)
		assert.Equal(t, resp, result)
	})

	t.Run("PollPayoutStatus", func(t *testing.T) {
		t.Parallel()
		// Given
		
		ctx := context.Background()
		req := models.PollPayoutStatusRequest{
			PayoutID: "test-payout",
		}
		resp := models.PollPayoutStatusResponse{
			Payment: nil,
			Error:   nil,
		}
		
		mockPlugin.EXPECT().PollPayoutStatus(ctx, req).Return(resp, nil)
		
		result, err := mockPlugin.PollPayoutStatus(ctx, req)
		// When/Then
		assert.NoError(t, err)
		assert.Equal(t, resp, result)
	})

	t.Run("PollTransferStatus", func(t *testing.T) {
		t.Parallel()
		// Given
		
		ctx := context.Background()
		req := models.PollTransferStatusRequest{
			TransferID: "test-transfer",
		}
		resp := models.PollTransferStatusResponse{
			Payment: nil,
			Error:   nil,
		}
		
		mockPlugin.EXPECT().PollTransferStatus(ctx, req).Return(resp, nil)
		
		result, err := mockPlugin.PollTransferStatus(ctx, req)
		// When/Then
		assert.NoError(t, err)
		assert.Equal(t, resp, result)
	})

	t.Run("ReversePayout", func(t *testing.T) {
		t.Parallel()
		// Given
		
		ctx := context.Background()
		req := models.ReversePayoutRequest{
			PaymentInitiationReversal: models.PSPPaymentInitiationReversal{},
		}
		resp := models.ReversePayoutResponse{
			Payment: models.PSPPayment{},
		}
		
		mockPlugin.EXPECT().ReversePayout(ctx, req).Return(resp, nil)
		
		result, err := mockPlugin.ReversePayout(ctx, req)
		// When/Then
		assert.NoError(t, err)
		assert.Equal(t, resp, result)
	})

	t.Run("ReverseTransfer", func(t *testing.T) {
		t.Parallel()
		// Given
		
		ctx := context.Background()
		req := models.ReverseTransferRequest{
			PaymentInitiationReversal: models.PSPPaymentInitiationReversal{},
		}
		resp := models.ReverseTransferResponse{
			Payment: models.PSPPayment{},
		}
		
		mockPlugin.EXPECT().ReverseTransfer(ctx, req).Return(resp, nil)
		
		result, err := mockPlugin.ReverseTransfer(ctx, req)
		// When/Then
		assert.NoError(t, err)
		assert.Equal(t, resp, result)
	})

	t.Run("TranslateWebhook", func(t *testing.T) {
		t.Parallel()
		// Given
		
		ctx := context.Background()
		req := models.TranslateWebhookRequest{
			Name:    "test-webhook",
			Webhook: models.PSPWebhook{},
		}
		resp := models.TranslateWebhookResponse{
			Responses: []models.WebhookResponse{},
		}
		
		mockPlugin.EXPECT().TranslateWebhook(ctx, req).Return(resp, nil)
		
		result, err := mockPlugin.TranslateWebhook(ctx, req)
		// When/Then
		assert.NoError(t, err)
		assert.Equal(t, resp, result)
	})
}
