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

func TestCreatePayout_Pending_ReturnsPollingID(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	now := time.Now().UTC()
	pi := models.PSPPaymentInitiation{
		Reference:   "test_payout_ref",
		Amount:      big.NewInt(1000), // $10.00 in cents
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
		Amount:               "10.00",
		Currency:             "USD",
		SourceAccountId:      "source_account_123",
		DestinationAccountId: "dest_account_456",
		Status:               "PENDING",
		CreatedAt:            now.Format(time.RFC3339),
		Metadata:             map[string]string{"test_key": "test_value"},
	}, nil)

	// When status is PENDING, CreatePayout should return PollingPayoutID
	resp, err := plugin.CreatePayout(context.Background(), models.CreatePayoutRequest{PaymentInitiation: pi})
	require.NoError(t, err)
	require.Nil(t, resp.Payment)
	require.NotNil(t, resp.PollingPayoutID)
	require.Equal(t, "payout_test_payout_ref", *resp.PollingPayoutID)
}

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
	}

	mockClient.EXPECT().CreatePayout(gomock.Any(), gomock.Any()).Return(&client.PayoutResponse{
		Id:                   "payout_test_payout_ref",
		IdempotencyKey:       "test_payout_ref",
		Amount:               "10.00",
		Currency:             "USD",
		SourceAccountId:      "source_account_123",
		DestinationAccountId: "dest_account_456",
		Status:               "SUCCEEDED",
		CreatedAt:            now.Format(time.RFC3339),
	}, nil)

	// When status is SUCCEEDED, CreatePayout should return Payment immediately
	resp, err := plugin.CreatePayout(context.Background(), models.CreatePayoutRequest{PaymentInitiation: pi})
	require.NoError(t, err)
	require.NotNil(t, resp.Payment)
	require.Nil(t, resp.PollingPayoutID)
	require.Equal(t, models.PAYMENT_STATUS_SUCCEEDED, resp.Payment.Status)
}

func TestPollPayoutStatus_Pending_ReturnsNil(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	now := time.Now().UTC()

	mockClient.EXPECT().GetPayoutStatus(gomock.Any(), "test_payout_id").Return(&client.PayoutResponse{
		Id:                   "test_payout_id",
		IdempotencyKey:       "test_payout_key",
		Amount:               "10.00",
		Currency:             "USD",
		SourceAccountId:      "source_account",
		DestinationAccountId: "dest_account",
		Status:               "PENDING",
		CreatedAt:            now.Format(time.RFC3339),
	}, nil)

	// When status is PENDING, PollPayoutStatus should return nil Payment (polling continues)
	resp, err := plugin.PollPayoutStatus(context.Background(), models.PollPayoutStatusRequest{PayoutID: "test_payout_id"})
	require.NoError(t, err)
	require.Nil(t, resp.Payment)
	require.Nil(t, resp.Error)
}

func TestPollPayoutStatus_Succeeded_ReturnsPayment(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	now := time.Now().UTC()

	mockClient.EXPECT().GetPayoutStatus(gomock.Any(), "test_payout_id").Return(&client.PayoutResponse{
		Id:                   "test_payout_id",
		IdempotencyKey:       "test_payout_key",
		Amount:               "10.00",
		Currency:             "USD",
		SourceAccountId:      "source_account",
		DestinationAccountId: "dest_account",
		Status:               "SUCCEEDED",
		CreatedAt:            now.Format(time.RFC3339),
	}, nil)

	// When status is SUCCEEDED, PollPayoutStatus should return Payment (polling stops)
	resp, err := plugin.PollPayoutStatus(context.Background(), models.PollPayoutStatusRequest{PayoutID: "test_payout_id"})
	require.NoError(t, err)
	require.NotNil(t, resp.Payment)
	require.Equal(t, models.PAYMENT_STATUS_SUCCEEDED, resp.Payment.Status)
	require.Equal(t, "test_payout_id", resp.Payment.Reference)
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
		Amount:               "10.00",
		Currency:             "USD",
		SourceAccountId:      "source_account_123",
		DestinationAccountId: "dest_account_456",
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

	mockClient.EXPECT().CreatePayout(gomock.Any(), gomock.Any()).Return(nil, errors.New("network error"))

	resp, err := plugin.CreatePayout(context.Background(), models.CreatePayoutRequest{PaymentInitiation: pi})
	require.Error(t, err)
	require.Contains(t, err.Error(), "network error")
	require.Nil(t, resp.Payment)
}

func TestCreatePayout_InvalidCurrency(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	pi := models.PSPPaymentInitiation{
		Reference:   "test_payout_ref",
		Amount:      big.NewInt(1000),
		Asset:       "INVALID/2",
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
}

func TestPollPayoutStatus_Failed_ReturnsPayment(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	now := time.Now().UTC()

	mockClient.EXPECT().GetPayoutStatus(gomock.Any(), "test_payout_id").Return(&client.PayoutResponse{
		Id:                   "test_payout_id",
		IdempotencyKey:       "test_payout_key",
		Amount:               "10.00",
		Currency:             "USD",
		SourceAccountId:      "source_account",
		DestinationAccountId: "dest_account",
		Status:               "FAILED",
		CreatedAt:            now.Format(time.RFC3339),
	}, nil)

	resp, err := plugin.PollPayoutStatus(context.Background(), models.PollPayoutStatusRequest{PayoutID: "test_payout_id"})
	require.NoError(t, err)
	require.NotNil(t, resp.Payment)
	require.Equal(t, models.PAYMENT_STATUS_FAILED, resp.Payment.Status)
}

func TestPollPayoutStatus_ClientError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	mockClient.EXPECT().GetPayoutStatus(gomock.Any(), "test_payout_id").Return(nil, errors.New("network error"))

	resp, err := plugin.PollPayoutStatus(context.Background(), models.PollPayoutStatusRequest{PayoutID: "test_payout_id"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "network error")
	require.Nil(t, resp.Payment)
}

func TestPayoutResponseToPayment(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	t.Run("valid response", func(t *testing.T) {
		resp := &client.PayoutResponse{
			Id:                   "payout_123",
			IdempotencyKey:       "idem_key",
			Amount:               "10.00",
			Currency:             "USD",
			SourceAccountId:      "source",
			DestinationAccountId: "dest",
			Status:               "SUCCEEDED",
			CreatedAt:            now.Format(time.RFC3339),
			Metadata:             map[string]string{"key": "value"},
		}

		payment, err := payoutResponseToPayment(resp, 2)
		require.NoError(t, err)
		require.Equal(t, "payout_123", payment.Reference)
		require.Equal(t, "idem_key", payment.ParentReference)
		require.Equal(t, models.PAYMENT_STATUS_SUCCEEDED, payment.Status)
		require.Equal(t, models.PAYMENT_TYPE_PAYOUT, payment.Type)
		require.Equal(t, big.NewInt(1000), payment.Amount)
	})

	t.Run("invalid amount", func(t *testing.T) {
		resp := &client.PayoutResponse{
			Id:                   "payout_123",
			IdempotencyKey:       "idem_key",
			Amount:               "invalid",
			Currency:             "USD",
			SourceAccountId:      "source",
			DestinationAccountId: "dest",
			Status:               "SUCCEEDED",
			CreatedAt:            now.Format(time.RFC3339),
		}

		_, err := payoutResponseToPayment(resp, 2)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse amount")
	})

	t.Run("invalid created at", func(t *testing.T) {
		resp := &client.PayoutResponse{
			Id:                   "payout_123",
			IdempotencyKey:       "idem_key",
			Amount:               "10.00",
			Currency:             "USD",
			SourceAccountId:      "source",
			DestinationAccountId: "dest",
			Status:               "SUCCEEDED",
			CreatedAt:            "invalid-date",
		}

		_, err := payoutResponseToPayment(resp, 2)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse created at")
	})
}

func TestAmountToString(t *testing.T) {
	t.Parallel()

	t.Run("standard USD amount", func(t *testing.T) {
		amount := big.NewInt(1000)
		result := amountToString(*amount, 2)
		require.Equal(t, "10.00", result)
	})

	t.Run("JPY amount with 0 precision", func(t *testing.T) {
		amount := big.NewInt(1000)
		result := amountToString(*amount, 0)
		require.Equal(t, "1000.", result)
	})

	t.Run("small amount", func(t *testing.T) {
		amount := big.NewInt(5)
		result := amountToString(*amount, 2)
		require.Equal(t, "0.05", result)
	})

	t.Run("very small amount", func(t *testing.T) {
		amount := big.NewInt(1)
		result := amountToString(*amount, 3)
		require.Equal(t, "0.001", result)
	})
}

func TestParseAmountFromString(t *testing.T) {
	t.Parallel()

	t.Run("decimal format", func(t *testing.T) {
		amount, err := parseAmountFromString("10.00", 2)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(1000), amount)
	})

	t.Run("integer format", func(t *testing.T) {
		amount, err := parseAmountFromString("1000", 2)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(1000), amount)
	})

	t.Run("decimal with fewer digits", func(t *testing.T) {
		amount, err := parseAmountFromString("10.5", 2)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(1050), amount)
	})

	t.Run("decimal with more digits truncated", func(t *testing.T) {
		amount, err := parseAmountFromString("10.999", 2)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(1099), amount)
	})

	t.Run("invalid decimal format", func(t *testing.T) {
		_, err := parseAmountFromString("10.00.00", 2)
		require.Error(t, err)
	})

	t.Run("invalid integer", func(t *testing.T) {
		_, err := parseAmountFromString("invalid", 2)
		require.Error(t, err)
	})
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

		err := plugin.validatePayoutRequest(pi)
		require.NoError(t, err)
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
