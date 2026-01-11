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

func TestCreateTransfer_Pending_ReturnsPollingID(t *testing.T) {
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
		Amount:               "10.00",
		Currency:             "USD",
		SourceAccountId:      "source_account_123",
		DestinationAccountId: "dest_account_456",
		Status:               "PENDING",
		CreatedAt:            now.Format(time.RFC3339),
		Metadata:             map[string]string{"test_key": "test_value"},
	}, nil)

	resp, err := plugin.CreateTransfer(context.Background(), models.CreateTransferRequest{PaymentInitiation: pi})
	require.NoError(t, err)
	require.Nil(t, resp.Payment)
	require.NotNil(t, resp.PollingTransferID)
	require.Equal(t, "transfer_test_transfer_ref", *resp.PollingTransferID)
}

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
	}

	mockClient.EXPECT().CreateTransfer(gomock.Any(), gomock.Any()).Return(&client.TransferResponse{
		Id:                   "transfer_test_transfer_ref",
		IdempotencyKey:       "test_transfer_ref",
		Amount:               "10.00",
		Currency:             "USD",
		SourceAccountId:      "source_account_123",
		DestinationAccountId: "dest_account_456",
		Status:               "SUCCEEDED",
		CreatedAt:            now.Format(time.RFC3339),
	}, nil)

	resp, err := plugin.CreateTransfer(context.Background(), models.CreateTransferRequest{PaymentInitiation: pi})
	require.NoError(t, err)
	require.NotNil(t, resp.Payment)
	require.Nil(t, resp.PollingTransferID)
	require.Equal(t, models.PAYMENT_STATUS_SUCCEEDED, resp.Payment.Status)
	require.Equal(t, models.PAYMENT_TYPE_TRANSFER, resp.Payment.Type)
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
		Amount:               "10.00",
		Currency:             "USD",
		SourceAccountId:      "source_account_123",
		DestinationAccountId: "dest_account_456",
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

	mockClient.EXPECT().CreateTransfer(gomock.Any(), gomock.Any()).Return(nil, errors.New("network error"))

	resp, err := plugin.CreateTransfer(context.Background(), models.CreateTransferRequest{PaymentInitiation: pi})
	require.Error(t, err)
	require.Contains(t, err.Error(), "network error")
	require.Nil(t, resp.Payment)
}

func TestCreateTransfer_InvalidCurrency(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	pi := models.PSPPaymentInitiation{
		Reference:   "test_transfer_ref",
		Amount:      big.NewInt(1000),
		Asset:       "INVALID/2",
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
}

func TestPollTransferStatus_Pending_ReturnsNil(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	now := time.Now().UTC()

	mockClient.EXPECT().GetTransferStatus(gomock.Any(), "test_transfer_id").Return(&client.TransferResponse{
		Id:                   "test_transfer_id",
		IdempotencyKey:       "test_transfer_key",
		Amount:               "10.00",
		Currency:             "USD",
		SourceAccountId:      "source_account",
		DestinationAccountId: "dest_account",
		Status:               "PENDING",
		CreatedAt:            now.Format(time.RFC3339),
	}, nil)

	resp, err := plugin.PollTransferStatus(context.Background(), models.PollTransferStatusRequest{TransferID: "test_transfer_id"})
	require.NoError(t, err)
	require.Nil(t, resp.Payment)
	require.Nil(t, resp.Error)
}

func TestPollTransferStatus_Succeeded_ReturnsPayment(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	now := time.Now().UTC()

	mockClient.EXPECT().GetTransferStatus(gomock.Any(), "test_transfer_id").Return(&client.TransferResponse{
		Id:                   "test_transfer_id",
		IdempotencyKey:       "test_transfer_key",
		Amount:               "10.00",
		Currency:             "USD",
		SourceAccountId:      "source_account",
		DestinationAccountId: "dest_account",
		Status:               "SUCCEEDED",
		CreatedAt:            now.Format(time.RFC3339),
	}, nil)

	resp, err := plugin.PollTransferStatus(context.Background(), models.PollTransferStatusRequest{TransferID: "test_transfer_id"})
	require.NoError(t, err)
	require.NotNil(t, resp.Payment)
	require.Equal(t, models.PAYMENT_STATUS_SUCCEEDED, resp.Payment.Status)
	require.Equal(t, "test_transfer_id", resp.Payment.Reference)
}

func TestPollTransferStatus_Failed_ReturnsPayment(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	now := time.Now().UTC()

	mockClient.EXPECT().GetTransferStatus(gomock.Any(), "test_transfer_id").Return(&client.TransferResponse{
		Id:                   "test_transfer_id",
		IdempotencyKey:       "test_transfer_key",
		Amount:               "10.00",
		Currency:             "USD",
		SourceAccountId:      "source_account",
		DestinationAccountId: "dest_account",
		Status:               "FAILED",
		CreatedAt:            now.Format(time.RFC3339),
	}, nil)

	resp, err := plugin.PollTransferStatus(context.Background(), models.PollTransferStatusRequest{TransferID: "test_transfer_id"})
	require.NoError(t, err)
	require.NotNil(t, resp.Payment)
	require.Equal(t, models.PAYMENT_STATUS_FAILED, resp.Payment.Status)
}

func TestPollTransferStatus_ClientError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	mockClient.EXPECT().GetTransferStatus(gomock.Any(), "test_transfer_id").Return(nil, errors.New("network error"))

	resp, err := plugin.PollTransferStatus(context.Background(), models.PollTransferStatusRequest{TransferID: "test_transfer_id"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "network error")
	require.Nil(t, resp.Payment)
}

func TestPollTransferStatus_UnknownCurrency_DefaultsPrecision(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := client.NewMockClient(ctrl)
	plugin := &Plugin{client: mockClient}

	now := time.Now().UTC()

	mockClient.EXPECT().GetTransferStatus(gomock.Any(), "test_transfer_id").Return(&client.TransferResponse{
		Id:                   "test_transfer_id",
		IdempotencyKey:       "test_transfer_key",
		Amount:               "10.00",
		Currency:             "UNKNOWN",
		SourceAccountId:      "source_account",
		DestinationAccountId: "dest_account",
		Status:               "SUCCEEDED",
		CreatedAt:            now.Format(time.RFC3339),
	}, nil)

	resp, err := plugin.PollTransferStatus(context.Background(), models.PollTransferStatusRequest{TransferID: "test_transfer_id"})
	require.NoError(t, err)
	require.NotNil(t, resp.Payment)
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

		err := plugin.validateTransferRequest(pi)
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

	t.Run("negative amount", func(t *testing.T) {
		pi := models.PSPPaymentInitiation{
			Reference: "test_ref",
			Amount:    big.NewInt(-100),
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

func TestTransferResponseToPayment(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	t.Run("valid response", func(t *testing.T) {
		resp := &client.TransferResponse{
			Id:                   "transfer_123",
			IdempotencyKey:       "idem_key",
			Amount:               "10.00",
			Currency:             "USD",
			SourceAccountId:      "source",
			DestinationAccountId: "dest",
			Status:               "SUCCEEDED",
			CreatedAt:            now.Format(time.RFC3339),
			Metadata:             map[string]string{"key": "value"},
		}

		payment, err := transferResponseToPayment(resp, 2)
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
			Currency:             "USD",
			SourceAccountId:      "source",
			DestinationAccountId: "dest",
			Status:               "SUCCEEDED",
			CreatedAt:            now.Format(time.RFC3339),
		}

		_, err := transferResponseToPayment(resp, 2)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse amount")
	})

	t.Run("invalid created at", func(t *testing.T) {
		resp := &client.TransferResponse{
			Id:                   "transfer_123",
			IdempotencyKey:       "idem_key",
			Amount:               "10.00",
			Currency:             "USD",
			SourceAccountId:      "source",
			DestinationAccountId: "dest",
			Status:               "SUCCEEDED",
			CreatedAt:            "invalid-date",
		}

		_, err := transferResponseToPayment(resp, 2)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse created at")
	})

	t.Run("JPY currency with 0 precision", func(t *testing.T) {
		resp := &client.TransferResponse{
			Id:                   "transfer_123",
			IdempotencyKey:       "idem_key",
			Amount:               "1000",
			Currency:             "JPY",
			SourceAccountId:      "source",
			DestinationAccountId: "dest",
			Status:               "SUCCEEDED",
			CreatedAt:            now.Format(time.RFC3339),
		}

		payment, err := transferResponseToPayment(resp, 0)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(1000), payment.Amount)
	})
}
