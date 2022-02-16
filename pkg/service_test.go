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
		_, err := service.CreatePayment(context.Background(), "test", payment.Data{
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

func TestUpdatePayment(t *testing.T) {
	runWithMock(t, func(t *mtest.T) {
		_, err := t.DB.Collection("Payment").InsertOne(context.Background(), map[string]interface{}{
			"organization": "foo",
			"_id":          "1",
		})
		assert.NoError(t, err)

		service := payment.NewDefaultService(t.DB)
		ret, err := service.UpdatePayment(context.Background(), "foo", "1", payment.Data{
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
		}, false)
		assert.NoError(t, err)
		assert.True(t, ret.Updated)
		assert.False(t, ret.Created)
	})
}

func TestUpsertPayment(t *testing.T) {
	runWithMock(t, func(t *mtest.T) {
		service := payment.NewDefaultService(t.DB)
		ret, err := service.UpdatePayment(context.Background(), "test", "1", payment.Data{
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
		}, true)
		assert.NoError(t, err)
		assert.True(t, ret.Created)
		assert.False(t, ret.Updated)
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
