package payment

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
)

type Service interface {
	CreatePayment(ctx context.Context, data PaymentData) (*Payment, error)
	UpdatePayment(ctx context.Context, id string, data PaymentData) error
	ListPayments(ctx context.Context) ([]*Payment, error)
}

type defaultServiceImpl struct {
	database *mongo.Database
}

func (d *defaultServiceImpl) ListPayments(ctx context.Context) ([]*Payment, error) {
	cursor, err := d.database.Collection("Payment").Find(ctx, map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	results := make([]*Payment, 0)
	err = cursor.All(ctx, &results)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (d *defaultServiceImpl) CreatePayment(ctx context.Context, data PaymentData) (*Payment, error) {
	payment := NewPayment(data)
	_, err := d.database.Collection("Payment").InsertOne(ctx, payment)
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

func (d *defaultServiceImpl) UpdatePayment(ctx context.Context, id string, data PaymentData) error {
	_, err := d.database.Collection("Payment").UpdateOne(ctx, map[string]interface{}{
		"_id": id,
	}, map[string]interface{}{
		"$set": data,
	})
	return err
}

var _ Service = &defaultServiceImpl{}

func NewDefaultService(database *mongo.Database) *defaultServiceImpl {
	return &defaultServiceImpl{
		database: database,
	}
}
