package database

import (
	"context"
	"log"
	"reflect"

	"github.com/formancehq/payments/internal/pkg/payments"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
)

func indexes() map[string][]mongo.IndexModel {
	return map[string][]mongo.IndexModel{
		payments.Collection: {
			{
				Keys: bsonx.Doc{
					bsonx.Elem{
						Key:   "provider",
						Value: bsonx.Int32(1),
					},
					bsonx.Elem{
						Key:   "reference",
						Value: bsonx.Int32(1),
					},
					bsonx.Elem{
						Key:   "type",
						Value: bsonx.Int32(1),
					},
				},
				Options: options.Index().SetUnique(true).SetName("identifier"),
			},
			{
				Keys: bsonx.Doc{
					bsonx.Elem{
						Key:   "provider",
						Value: bsonx.Int32(1),
					},
				},
				Options: options.Index().SetName("provider"),
			},
			{
				Keys: bsonx.Doc{
					bsonx.Elem{
						Key:   "type",
						Value: bsonx.Int32(1),
					},
				},
				Options: options.Index().SetName("payment-type"),
			},
			{
				Keys: bsonx.Doc{
					bsonx.Elem{
						Key:   "reference",
						Value: bsonx.Int32(1),
					},
				},
				Options: options.Index().SetName("payment-reference"),
			},
		},
	}
}

// TODO: Refactor
//
//nolint:gocyclo,cyclop // allow for now
func createIndexes(ctx context.Context, db *mongo.Database) error {
	type storedIndex struct {
		Unique                  bool               `bson:"unique"`
		ExpireAfterSeconds      int32              `bson:"expireAfterSeconds"`
		Name                    string             `bson:"name"`
		Key                     bsonx.Doc          `bson:"key"`
		Collation               *options.Collation `bson:"collation"`
		PartialFilterExpression interface{}        `bson:"partialFilterExpression"`
	}

	const id = "_id_"

	for entity, indexes := range indexes() {
		c := db.Collection(entity)

		listCursor, err := c.Indexes().List(ctx)
		if err != nil {
			return err
		}

		storedIndexes := make([]storedIndex, 0)

		err = listCursor.All(ctx, &storedIndexes)
		if err != nil {
			return err
		}

	l:
		for _, storedIndex := range storedIndexes {
			if storedIndex.Name == id {
				continue l
			}

			for _, index := range indexes {
				if *index.Options.Name != storedIndex.Name {
					continue
				}

				var modified bool
				if !reflect.DeepEqual(index.Keys, storedIndex.Key) {
					log.Printf("Keys of index %s of collection %s modified\r\n", *index.Options.Name, entity)
					modified = true
				}

				if (index.Options.PartialFilterExpression == nil && storedIndex.PartialFilterExpression != nil) ||
					(index.Options.PartialFilterExpression != nil && storedIndex.PartialFilterExpression == nil) ||
					!reflect.DeepEqual(index.Options.PartialFilterExpression, storedIndex.PartialFilterExpression) {
					log.Printf("PartialFilterExpression of index %s of collection %s modified\r\n", *index.Options.Name, entity)
					modified = true
				}

				if (index.Options.Unique == nil && storedIndex.Unique) ||
					(index.Options.Unique != nil && *index.Options.Unique != storedIndex.Unique) {
					log.Printf("Uniqueness of index %s of collection %s modified\r\n", *index.Options.Name, entity)
					modified = true
				}

				if (index.Options.ExpireAfterSeconds == nil && storedIndex.ExpireAfterSeconds > 0) ||
					(index.Options.ExpireAfterSeconds != nil && *index.Options.ExpireAfterSeconds != storedIndex.ExpireAfterSeconds) {
					log.Printf("ExpireAfterSeconds of index %s of collection %s modified\r\n", *index.Options.Name, entity)
					modified = true
				}

				if (index.Options.Collation == nil && storedIndex.Collation != nil) ||
					(index.Options.Collation != nil && storedIndex.Collation == nil) ||
					!reflect.DeepEqual(index.Options.Collation, storedIndex.Collation) {
					log.Printf("Collation of index %s of collection %s modified\r\n", *index.Options.Name, entity)
					modified = true
				}

				if !modified {
					log.Printf("Index %s of collection %s not modified\r\n", *index.Options.Name, entity)

					continue l
				}

				log.Printf("Recreate index %s on collection %s\r\n", *index.Options.Name, entity)

				_, err = c.Indexes().DropOne(ctx, storedIndex.Name)
				if err != nil {
					log.Printf("Unable to drop index %s of collection %s: %s\r\n", *index.Options.Name, entity, err)

					continue l
				}

				_, err = c.Indexes().CreateOne(ctx, index)
				if err != nil {
					log.Printf("Unable to create index %s of collection %s: %s\r\n", *index.Options.Name, entity, err)

					continue l
				}
			}
		}

		// Check for deleted index
	l3:
		for _, storedIndex := range storedIndexes {
			if storedIndex.Name == id {
				continue l3
			}

			for _, index := range indexes {
				if *index.Options.Name == storedIndex.Name {
					continue l3
				}
			}

			log.Printf("Detected deleted index %s on collection %s\r\n", storedIndex.Name, entity)

			_, err = c.Indexes().DropOne(ctx, storedIndex.Name)
			if err != nil {
				log.Printf("Unable to drop index %s of collection %s: %s\r\n", storedIndex.Name, entity, err)
			}
		}

		// Check for new indexes to create
	l2:
		for _, index := range indexes {
			for _, storedIndex := range storedIndexes {
				if *index.Options.Name == storedIndex.Name {
					continue l2
				}
			}

			log.Printf("Create new index %s on collection %s\r\n", *index.Options.Name, entity)

			_, err = c.Indexes().CreateOne(ctx, index)
			if err != nil {
				log.Printf("Unable to create index %s of collection %s: %s\r\n", *index.Options.Name, entity, err)
			}
		}
	}

	return nil
}
