package payment

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const Collection = "Payment"

type Sort struct {
	Key  string
	Desc bool
}

type ListQueryParameters struct {
	Limit int64
	Skip  int64
	Sort  []Sort
}

type UpdateResult struct {
	Found   bool
	Created bool
	Updated bool
}

type Service interface {
	SavePayment(ctx context.Context, data Payment) error
	ListPayments(ctx context.Context, parameters ListQueryParameters) (*mongo.Cursor, error)
}

type defaultServiceImpl struct {
	database  *mongo.Database
	publisher message.Publisher
}

func (d *defaultServiceImpl) ListPayments(ctx context.Context, parameters ListQueryParameters) (*mongo.Cursor, error) {
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
	pipeline := []bson.M{
		{
			"$sort": bson.M{
				"date": -1,
			},
		},
		{
			"$group": bson.M{
				"_id": "$id",
				"first": bson.M{
					"$first": "$$ROOT",
				},
			},
		},
		{
			"$replaceRoot": bson.M{
				"newRoot": "$first",
			},
		},
	}
	if parameters.Skip > 0 {
		pipeline = append(pipeline, bson.M{
			"$skip": parameters.Skip,
		})
	}
	if parameters.Limit > 0 {
		pipeline = append(pipeline, bson.M{
			"$limit": parameters.Limit,
		})
	}
	return d.database.Collection(Collection).Aggregate(ctx, pipeline)
}

func (d *defaultServiceImpl) SavePayment(ctx context.Context, p Payment) error {
	_, err := d.database.Collection(Collection).InsertOne(ctx, p)
	if err != nil {
		return err
	}
	if d.publisher != nil {
		err = d.publisher.Publish(TopicSavedPayment, newMessage(ctx, SavedPaymentEvent{
			Payment: p,
		}))
		if err != nil {
			logrus.Errorf("publishing created payment event to topic '%s': %s", TopicSavedPayment, err)
		}
	}
	return nil
}

var _ Service = &defaultServiceImpl{}

type serviceOption interface {
	apply(impl *defaultServiceImpl)
}
type ServiceOptionFn func(impl *defaultServiceImpl)

func (fn ServiceOptionFn) apply(impl *defaultServiceImpl) {
	fn(impl)
}

func WithPublisher(publisher message.Publisher) ServiceOptionFn {
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
