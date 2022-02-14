package payment_test

import (
	"context"
	payment "github.com/numary/payment/pkg"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"testing"
	"time"
)

func TestCreatePayment(t *testing.T) {
	runWithMock(t, "CreatePayment", func(t *mtest.T) {
		t.AddMockResponses(mtest.CreateSuccessResponse())

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
	runWithMock(t, "UpdatePayment", func(t *mtest.T) {
		t.AddMockResponses(mtest.CreateSuccessResponse())

		service := payment.NewDefaultService(t.DB)
		err := service.UpdatePayment(context.Background(), "test", uuid.New(), payment.Data{
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
	runWithMock(t, "ListPayments", func(t *mtest.T) {
		t.AddMockResponses(mtest.CreateCursorResponse(0, t.Name()+".Payment", mtest.FirstBatch, bson.D{
			{
				Key:   "_id",
				Value: uuid.New(),
			},
		}, bson.D{
			{
				Key:   "_id",
				Value: uuid.New(),
			},
		}, bson.D{
			{
				Key:   "_id",
				Value: uuid.New(),
			},
		}))

		service := payment.NewDefaultService(t.DB)
		cursor, err := service.ListPayments(context.Background(), "test", payment.ListQueryParameters{})
		assert.NoError(t, err)
		assert.Equal(t, cursor.RemainingBatchLength(), 3)
	})
}
