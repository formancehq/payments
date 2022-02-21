package payment_test

import (
	"context"
	payment "github.com/numary/payment/pkg"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"testing"
	"time"
)

func TestCreatePayment(t *testing.T) {
	runWithMock(t, func(t *mtest.T) {
		service := payment.NewDefaultService(t.DB)
		created, err := service.SavePayment(context.Background(), "test", payment.Data{
			ID:        "payment0",
			Provider:  "stripe",
			Reference: "ref",
			Scheme:    payment.SchemeSepa,
			Type:      payment.TypePayIn,
			Status:    "accepted",
			Value: payment.Value{
				Amount: 100,
				Asset:  "USD",
			},
			Date: time.Now(),
		})
		assert.NoError(t, err)
		assert.True(t, created)
	})
}

func TestListPayments(t *testing.T) {
	runWithMock(t, func(t *mtest.T) {
		t.DB.Collection("Payment").InsertMany(context.Background(), []interface{}{
			map[string]interface{}{
				"_id":          uuid.New(),
				"organization": "test",
			},
			map[string]interface{}{
				"_id":          uuid.New(),
				"organization": "test",
			},
			map[string]interface{}{
				"_id":          uuid.New(),
				"organization": "test",
			},
		})
		service := payment.NewDefaultService(t.DB)
		cursor, err := service.ListPayments(context.Background(), "test", payment.ListQueryParameters{})
		assert.NoError(t, err)
		assert.Equal(t, cursor.RemainingBatchLength(), 3)
	})
}
