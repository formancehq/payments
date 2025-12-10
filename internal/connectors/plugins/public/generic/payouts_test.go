package generic

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/require"
)

func testPlugin(t *testing.T) *Plugin {
	logger := logging.NewDefaultLogger(nil, true, false, false)
	config := json.RawMessage(`{"apiKey": "test", "endpoint": "https://api.example.com"}`)
	plugin, err := New("generic-test", logger, config)
	require.NoError(t, err)
	return plugin
}

func TestCreatePayout(t *testing.T) {
	t.Parallel()

	plugin := testPlugin(t)

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

	plugin := testPlugin(t)

	payment, err := plugin.pollPayoutStatus(context.Background(), "test_payout_id")
	require.NoError(t, err)

	require.Equal(t, models.PAYMENT_TYPE_PAYOUT, payment.Type)
	require.Equal(t, "test_payout_id", payment.Reference)
	require.Equal(t, models.PAYMENT_STATUS_SUCCEEDED, payment.Status)
}

func TestValidatePayoutRequest(t *testing.T) {
	t.Parallel()

	plugin := testPlugin(t)

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