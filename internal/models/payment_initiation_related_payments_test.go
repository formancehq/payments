package models_test

import (
	"testing"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPaymentInitiationRelatedPaymentsIdempotencyKey(t *testing.T) {
	t.Parallel()

	connectorID := models.ConnectorID{
		Provider:  "test",
		Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}
	
	paymentInitiationID := models.PaymentInitiationID{
		Reference:   "pi123",
		ConnectorID: connectorID,
	}
	
	paymentID := models.PaymentID{
		PaymentReference: models.PaymentReference{
			Reference: "payment123",
			Type:      models.PAYMENT_TYPE_PAYIN,
		},
		ConnectorID: connectorID,
	}
	
	relatedPayments := models.PaymentInitiationRelatedPayments{
		PaymentInitiationID: paymentInitiationID,
		PaymentID:           paymentID,
	}

	key := relatedPayments.IdempotencyKey()
	assert.Equal(t, models.IdempotencyKey(&relatedPayments), key)
}
