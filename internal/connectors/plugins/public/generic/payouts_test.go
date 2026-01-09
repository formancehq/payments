package generic

import (
	"context"
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
