package payment

import (
	"context"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"testing"
	"time"
)

func runWithMock(t *testing.T, name string, fn func(t *mtest.T)) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run(name, fn)
}

func TestCreatePayment(t *testing.T) {
	runWithMock(t, "CreatePayment", func(t *mtest.T) {
		t.AddMockResponses(mtest.CreateSuccessResponse())

		service := NewDefaultService(t.DB)
		_, err := service.CreatePayment(context.Background(), PaymentData{
			Provider:  "stripe",
			Reference: "ref",
			Scheme:    SchemeSepa,
			Type:      TypePayIn,
			Status:    "accepted",
			Value: PaymentValue{
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

		service := NewDefaultService(t.DB)
		err := service.UpdatePayment(context.Background(), uuid.New(), PaymentData{
			Provider:  "stripe",
			Reference: "ref",
			Scheme:    SchemeSepa,
			Type:      TypePayIn,
			Status:    "accepted",
			Value: PaymentValue{
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

		service := NewDefaultService(t.DB)
		payments, err := service.ListPayments(context.Background())
		assert.NoError(t, err)
		assert.Len(t, payments, 3)
	})
}
