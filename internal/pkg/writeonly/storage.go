package writeonly

import (
	"context"

	"github.com/iancoleman/strcase"
	"github.com/numary/payments/internal/pkg/payments"
	"go.mongodb.org/mongo-driver/mongo"
)

type Storage interface {
	Write(ctx context.Context, items ...any) error
}
type StorageFn func(ctx context.Context, items ...any) error

func (fn StorageFn) Write(ctx context.Context, items ...any) error {
	return fn(ctx, items...)
}

var NoOpStorage = StorageFn(func(ctx context.Context, items ...any) error {
	return nil
})

func Write[T any](ctx context.Context, storage Storage, items ...T) error {
	m := make([]any, 0)
	for _, item := range items {
		m = append(m, item)
	}
	return storage.Write(ctx, m...)
}

type MongoDBStorage struct {
	db             *mongo.Database
	provider       string
	taskDescriptor payments.TaskDescriptor
}

func (m *MongoDBStorage) Write(ctx context.Context, items ...any) error {
	toSave := make([]any, 0)
	for _, i := range items {
		toSave = append(toSave, Item{
			Provider: m.provider,
			TaskId:   payments.IDFromDescriptor(m.taskDescriptor),
			Data:     i,
		})
	}
	collectionName := strcase.ToCamel(m.provider) + "Storage"
	_, err := m.db.Collection(collectionName).InsertMany(ctx, toSave)
	if err != nil {
		return err
	}

	return nil
}

func NewMongoDBStorage(db *mongo.Database, provider string, descriptor payments.TaskDescriptor) *MongoDBStorage {
	return &MongoDBStorage{
		db:             db,
		provider:       provider,
		taskDescriptor: descriptor,
	}
}

var _ Storage = &MongoDBStorage{}
