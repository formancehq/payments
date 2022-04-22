package payment_test

import (
	"context"
	payment "github.com/numary/payments/pkg"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"testing"
	"time"
)

func TestCreatePayment(t *testing.T) {
	runWithMock(t, func(t *mtest.T) {
		service := payment.NewDefaultService(t.DB)
		err := service.SavePayment(context.Background(), payment.Payment{
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
	})
}

func TestListPayments(t *testing.T) {
	runWithMock(t, func(t *mtest.T) {
		_, err := t.DB.Collection(payment.Collection).InsertMany(context.Background(), []interface{}{
			map[string]interface{}{
				"id": uuid.New(),
			},
			map[string]interface{}{
				"id": uuid.New(),
			},
			map[string]interface{}{
				"id": uuid.New(),
			},
		})
		assert.NoError(t, err)

		service := payment.NewDefaultService(t.DB)
		cursor, err := service.ListPayments(context.Background(), payment.ListQueryParameters{})
		assert.NoError(t, err)
		assert.Equal(t, cursor.RemainingBatchLength(), 3)
	})
}

func TestPaymentHistory(t *testing.T) {
	runWithMock(t, func(t *mtest.T) {
		id := uuid.New()
		moreRecent := time.Now().Round(time.Second).UTC()
		_, err := t.DB.Collection(payment.Collection).InsertMany(context.Background(), []interface{}{
			map[string]interface{}{
				"id":             id,
				"organizationId": "test",
				"date":           moreRecent.Add(-2 * time.Minute),
			},
			map[string]interface{}{
				"id":             id,
				"organizationId": "test",
				"date":           moreRecent,
			},
			map[string]interface{}{
				"id":             id,
				"organizationId": "test",
				"date":           moreRecent.Add(-time.Minute),
			},
		})
		if !assert.NoError(t, err) {
			return
		}

		service := payment.NewDefaultService(t.DB)
		cursor, err := service.ListPayments(context.Background(), payment.ListQueryParameters{})
		if !assert.NoError(t, err) {
			return
		}
		if !assert.Equal(t, cursor.RemainingBatchLength(), 1) {
			return
		}
		if !assert.True(t, cursor.Next(context.Background())) {
			return
		}
		nextValue := payment.Payment{}
		if !assert.NoError(t, cursor.Decode(&nextValue)) {
			return
		}
		if !assert.Equal(t, moreRecent, nextValue.Date) {
			return
		}
	})
}
