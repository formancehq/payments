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

func TestCreatePayout(t *testing.T) {
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

	payment, err := plugin.createPayout(context.Background(), pi)
	require.NoError(t, err)

	require.Equal(t, models.PAYMENT_TYPE_PAYOUT, payment.Type)
	require.Equal(t, "test_payout_ref", payment.ParentReference)
	require.Equal(t, "payout_test_payout_ref", payment.Reference)
	require.Equal(t, models.PAYMENT_STATUS_PENDING, payment.Status)
	require.Equal(t, big.NewInt(1000), payment.Amount)
	require.Equal(t, "USD/2", payment.Asset)
	require.Equal(t, "source_account_123", *payment.SourceAccountReference)
	require.Equal(t, "dest_account_456", *payment.DestinationAccountReference)
	require.Equal(t, "test_value", payment.Metadata["test_key"])
}

func TestPollPayoutStatus(t *testing.T) {
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
		Metadata:             make(map[string]string),
	}, nil)

	payment, err := plugin.pollPayoutStatus(context.Background(), "test_payout_id")
	require.NoError(t, err)

	require.Equal(t, models.PAYMENT_TYPE_PAYOUT, payment.Type)
	require.Equal(t, "test_payout_id", payment.Reference)
	require.Equal(t, models.PAYMENT_STATUS_SUCCEEDED, payment.Status)
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