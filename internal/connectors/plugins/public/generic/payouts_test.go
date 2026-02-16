package generic

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/generic/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreatePayout_Succeeded_ReturnsPayment(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	now := time.Now().UTC()
	pi := models.PSPPaymentInitiation{
		Reference:   "test_payout_ref",
		Amount:      big.NewInt(1000),
		Asset:       "USD/2",
		Description: "Test payout",
		SourceAccount: &models.PSPAccount{
			Reference: "source_account_123",
		},
		DestinationAccount: &models.PSPAccount{
			Reference: "dest_account_456",
		},
		CreatedAt: now,
		Metadata: map[string]string{
			"test_key": "test_value",
		},
	}

	mockClient.EXPECT().CreatePayout(gomock.Any(), gomock.Any()).Return(&client.PayoutResponse{
		Id:                   "payout_test_payout_ref",
		IdempotencyKey:       "test_payout_ref",
		Amount:               "1000",
		Currency:             "USD/2",
		SourceAccountID:      "source_account_123",
		DestinationAccountID: "dest_account_456",
		Status:               "SUCCEEDED",
		CreatedAt:            now.Format(time.RFC3339),
		Metadata:             map[string]string{"test_key": "test_value"},
	}, nil)

	resp, err := plugin.CreatePayout(context.Background(), models.CreatePayoutRequest{PaymentInitiation: pi})
	require.NoError(t, err)
	require.NotNil(t, resp.Payment)
	require.Equal(t, models.PAYMENT_STATUS_SUCCEEDED, resp.Payment.Status)
	require.Equal(t, "payout_test_payout_ref", resp.Payment.Reference)
}

func TestCreatePayout_Pending_ReturnsPayment(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	now := time.Now().UTC()
	pi := models.PSPPaymentInitiation{
		Reference:   "test_payout_ref",
		Amount:      big.NewInt(1000),
		Asset:       "USD/2",
		Description: "Test payout",
		SourceAccount: &models.PSPAccount{
			Reference: "source_account_123",
		},
		DestinationAccount: &models.PSPAccount{
			Reference: "dest_account_456",
		},
		CreatedAt: now,
	}

	mockClient.EXPECT().CreatePayout(gomock.Any(), gomock.Any()).Return(&client.PayoutResponse{
		Id:                   "payout_test_payout_ref",
		IdempotencyKey:       "test_payout_ref",
		Amount:               "1000",
		Currency:             "USD/2",
		SourceAccountID:      "source_account_123",
		DestinationAccountID: "dest_account_456",
		Status:               "PENDING",
		CreatedAt:            now.Format(time.RFC3339),
	}, nil)

	resp, err := plugin.CreatePayout(context.Background(), models.CreatePayoutRequest{PaymentInitiation: pi})
	require.NoError(t, err)
	require.NotNil(t, resp.Payment)
	require.Equal(t, models.PAYMENT_STATUS_PENDING, resp.Payment.Status)
}

func TestCreatePayout_Failed_ReturnsPayment(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	now := time.Now().UTC()
	pi := models.PSPPaymentInitiation{
		Reference:   "test_payout_ref",
		Amount:      big.NewInt(1000),
		Asset:       "USD/2",
		Description: "Test payout",
		SourceAccount: &models.PSPAccount{
			Reference: "source_account_123",
		},
		DestinationAccount: &models.PSPAccount{
			Reference: "dest_account_456",
		},
		CreatedAt: now,
	}

	mockClient.EXPECT().CreatePayout(gomock.Any(), gomock.Any()).Return(&client.PayoutResponse{
		Id:                   "payout_test_payout_ref",
		IdempotencyKey:       "test_payout_ref",
		Amount:               "1000",
		Currency:             "USD/2",
		SourceAccountID:      "source_account_123",
		DestinationAccountID: "dest_account_456",
		Status:               "FAILED",
		CreatedAt:            now.Format(time.RFC3339),
	}, nil)

	resp, err := plugin.CreatePayout(context.Background(), models.CreatePayoutRequest{PaymentInitiation: pi})
	require.NoError(t, err)
	require.NotNil(t, resp.Payment)
	require.Equal(t, models.PAYMENT_STATUS_FAILED, resp.Payment.Status)
}

func TestCreatePayout_ClientError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	pi := models.PSPPaymentInitiation{
		Reference:   "test_payout_ref",
		Amount:      big.NewInt(1000),
		Asset:       "USD/2",
		Description: "Test payout",
		SourceAccount: &models.PSPAccount{
			Reference: "source_account_123",
		},
		DestinationAccount: &models.PSPAccount{
			Reference: "dest_account_456",
		},
	}

	mockClient.EXPECT().CreatePayout(gomock.Any(), gomock.Any()).Return(nil, errors.New("network error"))

	resp, err := plugin.CreatePayout(context.Background(), models.CreatePayoutRequest{PaymentInitiation: pi})
	require.Error(t, err)
	require.Contains(t, err.Error(), "network error")
	require.Nil(t, resp.Payment)
}

func TestCreatePayout_InvalidAssetFormat(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	pi := models.PSPPaymentInitiation{
		Reference:   "test_payout_ref",
		Amount:      big.NewInt(1000),
		Asset:       "USD/2/3",
		Description: "Test payout",
		SourceAccount: &models.PSPAccount{
			Reference: "source_account_123",
		},
		DestinationAccount: &models.PSPAccount{
			Reference: "dest_account_456",
		},
	}

	resp, err := plugin.CreatePayout(context.Background(), models.CreatePayoutRequest{PaymentInitiation: pi})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid asset format")
	require.Nil(t, resp.Payment)
}

func TestCreatePayout_NilAmount(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	pi := models.PSPPaymentInitiation{
		Reference:   "test_payout_ref",
		Amount:      nil,
		Asset:       "USD/2",
		Description: "Test payout",
		SourceAccount: &models.PSPAccount{
			Reference: "source_account_123",
		},
		DestinationAccount: &models.PSPAccount{
			Reference: "dest_account_456",
		},
	}

	resp, err := plugin.CreatePayout(context.Background(), models.CreatePayoutRequest{PaymentInitiation: pi})
	require.Error(t, err)
	require.Nil(t, resp.Payment)
	require.Contains(t, err.Error(), "amount must be positive")
}

func TestPayoutResponseToPayment(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	t.Run("valid response", func(t *testing.T) {
		resp := &client.PayoutResponse{
			Id:                   "payout_123",
			IdempotencyKey:       "idem_key",
			Amount:               "1000",
			Currency:             "USD/2",
			SourceAccountID:      "source",
			DestinationAccountID: "dest",
			Status:               "SUCCEEDED",
			CreatedAt:            now.Format(time.RFC3339),
			Metadata:             map[string]string{"key": "value"},
		}

		payment, err := payoutResponseToPayment(resp)
		require.NoError(t, err)
		require.Equal(t, "payout_123", payment.Reference)
		require.Equal(t, "idem_key", payment.ParentReference)
		require.Equal(t, models.PAYMENT_STATUS_SUCCEEDED, payment.Status)
		require.Equal(t, models.PAYMENT_TYPE_PAYOUT, payment.Type)
		require.Equal(t, big.NewInt(1000), payment.Amount)
		require.Equal(t, "USD/2", payment.Asset)
	})

	t.Run("invalid amount", func(t *testing.T) {
		resp := &client.PayoutResponse{
			Id:                   "payout_123",
			IdempotencyKey:       "idem_key",
			Amount:               "invalid",
			Currency:             "USD/2",
			SourceAccountID:      "source",
			DestinationAccountID: "dest",
			Status:               "SUCCEEDED",
			CreatedAt:            now.Format(time.RFC3339),
		}

		_, err := payoutResponseToPayment(resp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse amount")
	})

	t.Run("invalid createdAt", func(t *testing.T) {
		resp := &client.PayoutResponse{
			Id:                   "payout_123",
			IdempotencyKey:       "idem_key",
			Amount:               "1000",
			Currency:             "USD/2",
			SourceAccountID:      "source",
			DestinationAccountID: "dest",
			Status:               "SUCCEEDED",
			CreatedAt:            "invalid-date",
		}

		_, err := payoutResponseToPayment(resp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse createdAt")
	})

	t.Run("unknown status maps to OTHER", func(t *testing.T) {
		resp := &client.PayoutResponse{
			Id:                   "payout_123",
			IdempotencyKey:       "idem_key",
			Amount:               "1000",
			Currency:             "USD/2",
			SourceAccountID:      "source",
			DestinationAccountID: "dest",
			Status:               "UNKNOWN_STATUS",
			CreatedAt:            now.Format(time.RFC3339),
		}

		payment, err := payoutResponseToPayment(resp)
		require.NoError(t, err)
		require.Equal(t, models.PaymentStatus(models.PAYMENT_STATUS_OTHER), payment.Status)
	})
}

func TestMapStringToPaymentStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected models.PaymentStatus
	}{
		{"PENDING", models.PAYMENT_STATUS_PENDING},
		{"SUCCEEDED", models.PAYMENT_STATUS_SUCCEEDED},
		{"FAILED", models.PAYMENT_STATUS_FAILED},
		{"CANCELLED", models.PAYMENT_STATUS_CANCELLED},
		{"EXPIRED", models.PAYMENT_STATUS_EXPIRED},
		{"REFUNDED", models.PAYMENT_STATUS_REFUNDED},
		{"REFUNDED_FAILURE", models.PAYMENT_STATUS_REFUNDED_FAILURE},
		{"REFUND_REVERSED", models.PAYMENT_STATUS_REFUND_REVERSED},
		{"DISPUTE", models.PAYMENT_STATUS_DISPUTE},
		{"DISPUTE_WON", models.PAYMENT_STATUS_DISPUTE_WON},
		{"DISPUTE_LOST", models.PAYMENT_STATUS_DISPUTE_LOST},
		{"AUTHORISATION", models.PAYMENT_STATUS_AUTHORISATION},
		{"CAPTURE", models.PAYMENT_STATUS_CAPTURE},
		{"CAPTURE_FAILED", models.PAYMENT_STATUS_CAPTURE_FAILED},
		// Unknown/unsupported statuses map to OTHER
		{"PROCESSING", models.PAYMENT_STATUS_OTHER},
		{"UNKNOWN", models.PAYMENT_STATUS_OTHER},
		{"", models.PAYMENT_STATUS_OTHER},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.expected, mapStringToPaymentStatus(tc.input))
		})
	}
}

func TestValidatePayoutRequest(t *testing.T) {
	t.Parallel()

	plugin := &Plugin{}

	t.Run("valid request", func(t *testing.T) {
		pi := models.PSPPaymentInitiation{
			Reference: "test_ref",
			Amount:    big.NewInt(1000),
			SourceAccount: &models.PSPAccount{
				Reference: "source",
			},
			DestinationAccount: &models.PSPAccount{
				Reference: "dest",
			},
		}
		require.NoError(t, plugin.validatePayoutRequest(pi))
	})

	t.Run("missing source account", func(t *testing.T) {
		pi := models.PSPPaymentInitiation{
			Reference: "test_ref",
			Amount:    big.NewInt(1000),
			DestinationAccount: &models.PSPAccount{
				Reference: "dest",
			},
		}
		err := plugin.validatePayoutRequest(pi)
		require.Error(t, err)
		require.Contains(t, err.Error(), "source account is required")
	})

	t.Run("missing destination account", func(t *testing.T) {
		pi := models.PSPPaymentInitiation{
			Reference: "test_ref",
			Amount:    big.NewInt(1000),
			SourceAccount: &models.PSPAccount{
				Reference: "source",
			},
		}
		err := plugin.validatePayoutRequest(pi)
		require.Error(t, err)
		require.Contains(t, err.Error(), "destination account is required")
	})

	t.Run("invalid amount", func(t *testing.T) {
		pi := models.PSPPaymentInitiation{
			Reference: "test_ref",
			Amount:    big.NewInt(0),
			SourceAccount: &models.PSPAccount{
				Reference: "source",
			},
			DestinationAccount: &models.PSPAccount{
				Reference: "dest",
			},
		}
		err := plugin.validatePayoutRequest(pi)
		require.Error(t, err)
		require.Contains(t, err.Error(), "amount must be positive")
	})

	t.Run("missing reference", func(t *testing.T) {
		pi := models.PSPPaymentInitiation{
			Reference: "",
			Amount:    big.NewInt(1000),
			SourceAccount: &models.PSPAccount{
				Reference: "source",
			},
			DestinationAccount: &models.PSPAccount{
				Reference: "dest",
			},
		}
		err := plugin.validatePayoutRequest(pi)
		require.Error(t, err)
		require.Contains(t, err.Error(), "reference is required")
	})
}
