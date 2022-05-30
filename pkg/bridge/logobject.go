package bridge

import (
	"context"
	"fmt"
	"github.com/gobeam/stringy"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
)

type LogObjectStorage interface {
	Store(ctx context.Context, objects ...any) error
	drop(ctx context.Context) error
}

type noOpLogObjectStorage struct{}

func (n noOpLogObjectStorage) Store(ctx context.Context, objects ...any) error {
	return nil
}

func (n noOpLogObjectStorage) drop(ctx context.Context) error {
	return nil
}

var NoOpLogObjectStorage LogObjectStorage = &noOpLogObjectStorage{}

type defaultLogObjectStorage struct {
	db     *mongo.Database
	name   string
	logger sharedlogging.Logger
}

func (d *defaultLogObjectStorage) collection() string {
	return fmt.Sprintf("%sLogObjectStorage", stringy.New(d.name).CamelCase())
}

func (d *defaultLogObjectStorage) drop(ctx context.Context) error {
	err := d.db.Collection(d.collection()).Drop(ctx)
	if err != nil {
		return errors.Wrap(err, "removing LogObjectStorage")
	}
	d.logger.Infof("Log Object storage deleted")
	return nil
}

func (d defaultLogObjectStorage) Store(ctx context.Context, objects ...any) error {
	_, err := d.db.Collection(d.collection()).
		InsertMany(ctx, objects)
	return err
}

var _ LogObjectStorage = &defaultLogObjectStorage{}

func NewDefaultLogObjectStorage(name string, db *mongo.Database, logger sharedlogging.Logger) *defaultLogObjectStorage {
	return &defaultLogObjectStorage{
		db:     db,
		name:   name,
		logger: logger,
	}
}
