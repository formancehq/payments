package bridge

import (
	"context"
	"fmt"
	"github.com/gobeam/stringy"
	"go.mongodb.org/mongo-driver/mongo"
)

type LogObjectStorage interface {
	Store(ctx context.Context, objects ...any) error
}

type defaultLogObjectStorage struct {
	db   *mongo.Database
	name string
}

func (d defaultLogObjectStorage) Store(ctx context.Context, objects ...any) error {
	str := stringy.New(d.name)
	_, err := d.db.Collection(fmt.Sprintf("%sLogObjectStorage", str.CamelCase())).
		InsertMany(ctx, objects)
	return err
}

var _ LogObjectStorage = &defaultLogObjectStorage{}

func NewDefaultLogObjectStorage(name string, db *mongo.Database) *defaultLogObjectStorage {
	return &defaultLogObjectStorage{
		db:   db,
		name: name,
	}
}
