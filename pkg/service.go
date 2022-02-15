package payment

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const PaymentCollection = "Payment"

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
	UpdatePayment(ctx context.Context, organization string, id string, data Data) (bool, error)
	ListPayments(ctx context.Context, organization string, parameters ListQueryParameters) (*mongo.Cursor, error)
}

type defaultServiceImpl struct {
	database  *mongo.Database
	publisher message.Publisher
}

func (d *defaultServiceImpl) ListPayments(ctx context.Context, org string, parameters ListQueryParameters) (*mongo.Cursor, error) {
	opts := options.Find()
	if parameters.Skip != 0 {
		opts = opts.SetSkip(parameters.Skip)
	}
	if parameters.Limit != 0 {
		opts = opts.SetLimit(parameters.Limit)
	}
	if parameters.Sort != nil && len(parameters.Sort) > 0 {
		sort := bson.D{}
		for _, s := range parameters.Sort {
			sort = append(sort, bson.E{
				Key: s.Key,
				Value: func() int {
					if s.Desc {
						return -1
					}
					return 1
				}(),
			})
		}
		opts = opts.SetSort(sort)
	}
	return d.database.Collection(PaymentCollection).Find(ctx, map[string]interface{}{
		"organization": org,
	}, opts)
}

func (d *defaultServiceImpl) CreatePayment(ctx context.Context, org string, data Data) (*Payment, error) {
	payment := NewPayment(data)
	_, err := d.database.Collection(PaymentCollection).InsertOne(ctx, struct {
		Payment      `bson:",inline"`
		Organization string `bson:"organization"`
	}{
		Payment:      payment,
		Organization: org,
	})
	if err != nil {
		return nil, err
	}

	if d.publisher != nil {
		err = d.publisher.Publish(TopicCreatedPayment, newMessage(CreatedPaymentEvent{
			Payment: payment,
		}))
		if err != nil {
			logrus.Errorf("publishing created payment event to topic '%s': %s", TopicCreatedPayment, err)
		}
	}
	return &payment, nil
}

func (d *defaultServiceImpl) UpdatePayment(ctx context.Context, organization string, id string, data Data) (bool, error) {
	ret, err := d.database.Collection(PaymentCollection).UpdateOne(ctx, map[string]interface{}{
		"_id":          id,
		"organization": organization,
	}, map[string]interface{}{
		"$set": data,
	})
	if err != nil {
		return false, err
	}
	if ret.ModifiedCount == 0 {
		return false, nil
	}
	if d.publisher != nil {
		err = d.publisher.Publish(TopicUpdatedPayment, newMessage(UpdatedPaymentEvent{
			ID:   id,
			Data: data,
		}))
		if err != nil {
			logrus.Errorf("publishing created payment event to topic '%s': %s", TopicCreatedPayment, err)
		}
	}
	return true, err
}

var _ Service = &defaultServiceImpl{}

type serviceOption interface {
	apply(impl *defaultServiceImpl)
}
type serviceOptionFn func(impl *defaultServiceImpl)

func (fn serviceOptionFn) apply(impl *defaultServiceImpl) {
	fn(impl)
}

func WithPublisher(publisher message.Publisher) serviceOptionFn {
	return func(impl *defaultServiceImpl) {
		impl.publisher = publisher
	}
}

func NewDefaultService(database *mongo.Database, opts ...serviceOption) *defaultServiceImpl {
	ret := &defaultServiceImpl{
		database: database,
	}
	for _, opt := range opts {
		opt.apply(ret)
	}

	return ret
}
