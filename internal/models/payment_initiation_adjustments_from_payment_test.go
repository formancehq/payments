package models_test

import (
	"errors"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestFromPaymentToPaymentInitiationAdjustment(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Provider:  "test",
		Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}

	piID := models.PaymentInitiationID{
		Reference:   "pi123",
		ConnectorID: connectorID,
	}

	testCases := []struct {
		name           string
		paymentStatus  models.PaymentStatus
		expectedStatus models.PaymentInitiationAdjustmentStatus
		expectedError  error
		expectNil      bool
	}{
		{
			name:          "PAYMENT_STATUS_AMOUNT_ADJUSTMENT returns nil",
			paymentStatus: models.PAYMENT_STATUS_AMOUNT_ADJUSTMENT,
			expectNil:     true,
		},
		{
			name:          "PAYMENT_STATUS_UNKNOWN returns nil",
			paymentStatus: models.PAYMENT_STATUS_UNKNOWN,
			expectNil:     true,
		},
		{
			name:           "PAYMENT_STATUS_PENDING maps to PROCESSING",
			paymentStatus:  models.PAYMENT_STATUS_PENDING,
			expectedStatus: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING,
		},
		{
			name:           "PAYMENT_STATUS_AUTHORISATION maps to PROCESSING",
			paymentStatus:  models.PAYMENT_STATUS_AUTHORISATION,
			expectedStatus: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING,
		},
		{
			name:           "PAYMENT_STATUS_SUCCEEDED maps to PROCESSED",
			paymentStatus:  models.PAYMENT_STATUS_SUCCEEDED,
			expectedStatus: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
		},
		{
			name:           "PAYMENT_STATUS_CAPTURE maps to PROCESSED",
			paymentStatus:  models.PAYMENT_STATUS_CAPTURE,
			expectedStatus: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
		},
		{
			name:           "PAYMENT_STATUS_REFUND_REVERSED maps to PROCESSED",
			paymentStatus:  models.PAYMENT_STATUS_REFUND_REVERSED,
			expectedStatus: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
		},
		{
			name:           "PAYMENT_STATUS_DISPUTE_WON maps to PROCESSED",
			paymentStatus:  models.PAYMENT_STATUS_DISPUTE_WON,
			expectedStatus: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
		},
		{
			name:           "PAYMENT_STATUS_CANCELLED maps to FAILED",
			paymentStatus:  models.PAYMENT_STATUS_CANCELLED,
			expectedStatus: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED,
			expectedError:  errors.New("payment failed"),
		},
		{
			name:           "PAYMENT_STATUS_CAPTURE_FAILED maps to FAILED",
			paymentStatus:  models.PAYMENT_STATUS_CAPTURE_FAILED,
			expectedStatus: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED,
			expectedError:  errors.New("payment failed"),
		},
		{
			name:           "PAYMENT_STATUS_EXPIRED maps to FAILED",
			paymentStatus:  models.PAYMENT_STATUS_EXPIRED,
			expectedStatus: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED,
			expectedError:  errors.New("payment failed"),
		},
		{
			name:           "PAYMENT_STATUS_FAILED maps to FAILED",
			paymentStatus:  models.PAYMENT_STATUS_FAILED,
			expectedStatus: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED,
			expectedError:  errors.New("payment failed"),
		},
		{
			name:           "PAYMENT_STATUS_DISPUTE_LOST maps to FAILED",
			paymentStatus:  models.PAYMENT_STATUS_DISPUTE_LOST,
			expectedStatus: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED,
			expectedError:  errors.New("payment failed"),
		},
		{
			name:           "PAYMENT_STATUS_DISPUTE maps to UNKNOWN",
			paymentStatus:  models.PAYMENT_STATUS_DISPUTE,
			expectedStatus: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_UNKNOWN,
		},
		{
			name:           "PAYMENT_STATUS_REFUNDED maps to REVERSED",
			paymentStatus:  models.PAYMENT_STATUS_REFUNDED,
			expectedStatus: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSED,
		},
		{
			name:           "PAYMENT_STATUS_REFUNDED_FAILURE maps to REVERSE_FAILED",
			paymentStatus:  models.PAYMENT_STATUS_REFUNDED_FAILURE,
			expectedStatus: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_FAILED,
			expectedError:  errors.New("payment refund failed"),
		},
		{
			name:          "Unknown status returns nil",
			paymentStatus: 9999, // Invalid status
			expectNil:     true,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Given

			payment := &models.Payment{
				Status:    tc.paymentStatus,
				CreatedAt: now,
			}

			result := models.FromPaymentDataToPaymentInitiationAdjustment(payment.Status, payment.CreatedAt, piID)

			if tc.expectNil {

				assert.Nil(t, result)
				return
			}

			assert.NotNil(t, result)
			assert.Equal(t, tc.expectedStatus, result.Status)
			assert.Equal(t, now, result.CreatedAt)

			if tc.expectedError != nil {
				assert.NotNil(t, result.Error)
				assert.Equal(t, tc.expectedError.Error(), result.Error.Error())
			} else {
				assert.Nil(t, result.Error)
			}
		})
	}
}
