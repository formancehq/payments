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
	SavePayment(ctx context.Context, organization string, data Data) (bool, error)
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
	return d.database.Collection(Collection).Find(ctx, map[string]interface{}{
		"organizationId": org,
	}, opts)
}

func (d *defaultServiceImpl) SavePayment(ctx context.Context, org string, data Data) (bool, error) {
	ret, err := d.database.Collection(Collection).UpdateOne(ctx, map[string]interface{}{
		"_id":            data.ID,
		"organizationId": org,
	}, map[string]interface{}{
		"$set": data,
	}, options.Update().SetUpsert(true))
	if err != nil {
		return false, err
	}
	if d.publisher != nil {
		err = d.publisher.Publish(TopicSavedPayment, newMessage(ctx, SavedPaymentEvent{
			Payment: Payment{
				Data:           data,
				OrganizationID: org,
			},
		}))
		if err != nil {
			logrus.Errorf("publishing created payment event to topic '%s': %s", TopicSavedPayment, err)
		}
	}
	return ret.UpsertedCount > 0, nil
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
