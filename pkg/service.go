package payment

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Sort struct {
	Key  string
	Desc bool
}

type ListQueryParameters struct {
	Limit int64
	Skip  int64
	Sort  []Sort
}

type Service interface {
	CreatePayment(ctx context.Context, organization string, data Data) (*Payment, error)
	UpdatePayment(ctx context.Context, organization string, id string, data Data) error
	ListPayments(ctx context.Context, organization string, parameters ListQueryParameters) (*mongo.Cursor, error)
}

type defaultServiceImpl struct {
	database *mongo.Database
}

func (d *defaultServiceImpl) ListPayments(ctx context.Context, org string, parameters ListQueryParameters) (*mongo.Cursor, error) {
	opts := options.Find()
	if parameters.Skip != 0 {
		opts = opts.SetSkip(parameters.Skip)
	}
	if parameters.Limit != 0 {
		opts = opts.SetLimit(parameters.Limit)
	}
	if parameters.Sort != nil {
		for _, s := range parameters.Sort {
			opts = opts.SetSort(bson.E{
				Key: s.Key,
				Value: func() int {
					if s.Desc {
						return -1
					}
					return 1
				}(),
			})
		}
	}
	return d.database.Collection("Payment").Find(ctx, map[string]interface{}{
		"organization": org,
	}, opts)
}

func (d *defaultServiceImpl) CreatePayment(ctx context.Context, org string, data Data) (*Payment, error) {
	payment := NewPayment(data)
	_, err := d.database.Collection("Payment").InsertOne(ctx, struct {
		Payment      `bson:",inline"`
		Organization string `bson:"organization"`
	}{
		Payment:      payment,
		Organization: org,
	})
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

func (d *defaultServiceImpl) UpdatePayment(ctx context.Context, organization string, id string, data Data) error {
	_, err := d.database.Collection("Payment").UpdateOne(ctx, map[string]interface{}{
		"_id":          id,
		"organization": organization,
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
