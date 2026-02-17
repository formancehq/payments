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

func TestCreateTransfer_Succeeded_ReturnsPayment(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	now := time.Now().UTC()
	pi := models.PSPPaymentInitiation{
		Reference:   "test_transfer_ref",
		Amount:      big.NewInt(1000),
		Asset:       "USD/2",
		Description: "Test transfer",
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

	mockClient.EXPECT().CreateTransfer(gomock.Any(), gomock.Any()).Return(&client.TransferResponse{
		Id:                   "transfer_test_transfer_ref",
		IdempotencyKey:       "test_transfer_ref",
		Amount:               "1000",
		Currency:             "USD/2",
		SourceAccountID:      "source_account_123",
		DestinationAccountID: "dest_account_456",
		Status:               "SUCCEEDED",
		CreatedAt:            now.Format(time.RFC3339),
		Metadata:             map[string]string{"test_key": "test_value"},
	}, nil)

	resp, err := plugin.CreateTransfer(context.Background(), models.CreateTransferRequest{PaymentInitiation: pi})
	require.NoError(t, err)
	require.NotNil(t, resp.Payment)
	require.Equal(t, models.PAYMENT_STATUS_SUCCEEDED, resp.Payment.Status)
	require.Equal(t, models.PAYMENT_TYPE_TRANSFER, resp.Payment.Type)
	require.Equal(t, "transfer_test_transfer_ref", resp.Payment.Reference)
}

func TestCreateTransfer_Pending_ReturnsPayment(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	now := time.Now().UTC()
	pi := models.PSPPaymentInitiation{
		Reference:   "test_transfer_ref",
		Amount:      big.NewInt(1000),
		Asset:       "USD/2",
		Description: "Test transfer",
		SourceAccount: &models.PSPAccount{
			Reference: "source_account_123",
		},
		DestinationAccount: &models.PSPAccount{
			Reference: "dest_account_456",
		},
		CreatedAt: now,
	}

	mockClient.EXPECT().CreateTransfer(gomock.Any(), gomock.Any()).Return(&client.TransferResponse{
		Id:                   "transfer_test_transfer_ref",
		IdempotencyKey:       "test_transfer_ref",
		Amount:               "1000",
		Currency:             "USD/2",
		SourceAccountID:      "source_account_123",
		DestinationAccountID: "dest_account_456",
		Status:               "PENDING",
		CreatedAt:            now.Format(time.RFC3339),
	}, nil)

	resp, err := plugin.CreateTransfer(context.Background(), models.CreateTransferRequest{PaymentInitiation: pi})
	require.NoError(t, err)
	require.NotNil(t, resp.Payment)
	require.Equal(t, models.PAYMENT_STATUS_PENDING, resp.Payment.Status)
}

func TestCreateTransfer_Failed_ReturnsPayment(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	now := time.Now().UTC()
	pi := models.PSPPaymentInitiation{
		Reference:   "test_transfer_ref",
		Amount:      big.NewInt(1000),
		Asset:       "USD/2",
		Description: "Test transfer",
		SourceAccount: &models.PSPAccount{
			Reference: "source_account_123",
		},
		DestinationAccount: &models.PSPAccount{
			Reference: "dest_account_456",
		},
		CreatedAt: now,
	}

	mockClient.EXPECT().CreateTransfer(gomock.Any(), gomock.Any()).Return(&client.TransferResponse{
		Id:                   "transfer_test_transfer_ref",
		IdempotencyKey:       "test_transfer_ref",
		Amount:               "1000",
		Currency:             "USD/2",
		SourceAccountID:      "source_account_123",
		DestinationAccountID: "dest_account_456",
		Status:               "FAILED",
		CreatedAt:            now.Format(time.RFC3339),
	}, nil)

	resp, err := plugin.CreateTransfer(context.Background(), models.CreateTransferRequest{PaymentInitiation: pi})
	require.NoError(t, err)
	require.NotNil(t, resp.Payment)
	require.Equal(t, models.PAYMENT_STATUS_FAILED, resp.Payment.Status)
}

func TestCreateTransfer_ClientError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	pi := models.PSPPaymentInitiation{
		Reference:   "test_transfer_ref",
		Amount:      big.NewInt(1000),
		Asset:       "USD/2",
		Description: "Test transfer",
		SourceAccount: &models.PSPAccount{
			Reference: "source_account_123",
		},
		DestinationAccount: &models.PSPAccount{
			Reference: "dest_account_456",
		},
	}

	mockClient.EXPECT().CreateTransfer(gomock.Any(), gomock.Any()).Return(nil, errors.New("network error"))

	resp, err := plugin.CreateTransfer(context.Background(), models.CreateTransferRequest{PaymentInitiation: pi})
	require.Error(t, err)
	require.Contains(t, err.Error(), "network error")
	require.Nil(t, resp.Payment)
}

func TestCreateTransfer_InvalidAssetFormat(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	pi := models.PSPPaymentInitiation{
		Reference:   "test_transfer_ref",
		Amount:      big.NewInt(1000),
		Asset:       "USD/2/3",
		Description: "Test transfer",
		SourceAccount: &models.PSPAccount{
			Reference: "source_account_123",
		},
		DestinationAccount: &models.PSPAccount{
			Reference: "dest_account_456",
		},
	}

	resp, err := plugin.CreateTransfer(context.Background(), models.CreateTransferRequest{PaymentInitiation: pi})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid asset format")
	require.Nil(t, resp.Payment)
}

func TestCreateTransfer_NilAmount(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	pi := models.PSPPaymentInitiation{
		Reference:   "test_transfer_ref",
		Amount:      nil,
		Asset:       "USD/2",
		Description: "Test transfer",
		SourceAccount: &models.PSPAccount{
			Reference: "source_account_123",
		},
		DestinationAccount: &models.PSPAccount{
			Reference: "dest_account_456",
		},
	}

	resp, err := plugin.CreateTransfer(context.Background(), models.CreateTransferRequest{PaymentInitiation: pi})
	require.Error(t, err)
	require.Nil(t, resp.Payment)
	require.Contains(t, err.Error(), "amount must be positive")
}

func TestTransferResponseToPayment(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	t.Run("valid response", func(t *testing.T) {
		resp := &client.TransferResponse{
			Id:                   "transfer_123",
			IdempotencyKey:       "idem_key",
			Amount:               "1000",
			Currency:             "USD/2",
			SourceAccountID:      "source",
			DestinationAccountID: "dest",
			Status:               "SUCCEEDED",
			CreatedAt:            now.Format(time.RFC3339),
			Metadata:             map[string]string{"key": "value"},
		}

		payment, err := transferResponseToPayment(resp)
		require.NoError(t, err)
		require.Equal(t, "transfer_123", payment.Reference)
		require.Equal(t, "idem_key", payment.ParentReference)
		require.Equal(t, models.PAYMENT_STATUS_SUCCEEDED, payment.Status)
		require.Equal(t, models.PAYMENT_TYPE_TRANSFER, payment.Type)
		require.Equal(t, big.NewInt(1000), payment.Amount)
		require.Equal(t, "source", *payment.SourceAccountReference)
		require.Equal(t, "dest", *payment.DestinationAccountReference)
	})

	t.Run("invalid amount", func(t *testing.T) {
		resp := &client.TransferResponse{
			Id:                   "transfer_123",
			IdempotencyKey:       "idem_key",
			Amount:               "invalid",
			Currency:             "USD/2",
			SourceAccountID:      "source",
			DestinationAccountID: "dest",
			Status:               "SUCCEEDED",
			CreatedAt:            now.Format(time.RFC3339),
		}

		_, err := transferResponseToPayment(resp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse amount")
	})

	t.Run("invalid createdAt", func(t *testing.T) {
		resp := &client.TransferResponse{
			Id:                   "transfer_123",
			IdempotencyKey:       "idem_key",
			Amount:               "1000",
			Currency:             "USD/2",
			SourceAccountID:      "source",
			DestinationAccountID: "dest",
			Status:               "SUCCEEDED",
			CreatedAt:            "invalid-date",
		}

		_, err := transferResponseToPayment(resp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse createdAt")
	})

	t.Run("all status mappings", func(t *testing.T) {
		statuses := map[string]models.PaymentStatus{
			"PENDING":   models.PAYMENT_STATUS_PENDING,
			"SUCCEEDED": models.PAYMENT_STATUS_SUCCEEDED,
			"FAILED":    models.PAYMENT_STATUS_FAILED,
			"CANCELLED": models.PAYMENT_STATUS_CANCELLED,
			"OTHER":     models.PAYMENT_STATUS_OTHER,
		}

		for status, expected := range statuses {
			resp := &client.TransferResponse{
				Id:                   "transfer_123",
				IdempotencyKey:       "idem_key",
				Amount:               "1000",
				Currency:             "USD/2",
				SourceAccountID:      "source",
				DestinationAccountID: "dest",
				Status:               status,
				CreatedAt:            now.Format(time.RFC3339),
			}

			payment, err := transferResponseToPayment(resp)
			require.NoError(t, err)
			require.Equal(t, expected, payment.Status)
		}
	})
}

func TestValidateTransferRequest(t *testing.T) {
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
		require.NoError(t, plugin.validateTransferRequest(pi))
	})

	t.Run("missing source account", func(t *testing.T) {
		pi := models.PSPPaymentInitiation{
			Reference: "test_ref",
			Amount:    big.NewInt(1000),
			DestinationAccount: &models.PSPAccount{
				Reference: "dest",
			},
		}
		err := plugin.validateTransferRequest(pi)
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
		err := plugin.validateTransferRequest(pi)
		require.Error(t, err)
		require.Contains(t, err.Error(), "destination account is required")
	})

	t.Run("nil amount", func(t *testing.T) {
		pi := models.PSPPaymentInitiation{
			Reference: "test_ref",
			Amount:    nil,
			SourceAccount: &models.PSPAccount{
				Reference: "source",
			},
			DestinationAccount: &models.PSPAccount{
				Reference: "dest",
			},
		}
		err := plugin.validateTransferRequest(pi)
		require.Error(t, err)
		require.Contains(t, err.Error(), "amount must be positive")
	})

	t.Run("zero amount", func(t *testing.T) {
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
		err := plugin.validateTransferRequest(pi)
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
		err := plugin.validateTransferRequest(pi)
		require.Error(t, err)
		require.Contains(t, err.Error(), "reference is required")
	})
}
