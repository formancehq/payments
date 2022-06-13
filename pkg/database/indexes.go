package database

import (
	"context"
	"fmt"
	"github.com/numary/payments/pkg"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"reflect"
)

var indexes = map[string][]mongo.IndexModel{
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

func CreateIndexes(ctx context.Context, db *mongo.Database) error {

	type StoredIndex struct {
		Unique                  bool               `bson:"unique"`
		Key                     bsonx.Doc          `bson:"key"`
		Name                    string             `bson:"name"`
		PartialFilterExpression interface{}        `bson:"partialFilterExpression"`
		ExpireAfterSeconds      int32              `bson:"expireAfterSeconds"`
		Collation               *options.Collation `bson:"collation"`
	}

	for entity, indexes := range indexes {
		c := db.Collection(entity)
		listCursor, err := c.Indexes().List(ctx)
		if err != nil {
			return err
		}

		storedIndexes := make([]StoredIndex, 0)
		err = listCursor.All(ctx, &storedIndexes)
		if err != nil {
			return err
		}

	l:
		for _, storedIndex := range storedIndexes {
			if storedIndex.Name == "_id_" {
				continue l
			}
			for _, index := range indexes {
				if *index.Options.Name != storedIndex.Name {
					continue
				}
				var modified bool
				if !reflect.DeepEqual(index.Keys, storedIndex.Key) {
					fmt.Printf("Keys of index %s of collection %s modified\r\n", *index.Options.Name, entity)
					modified = true
				}
				if (index.Options.PartialFilterExpression == nil && storedIndex.PartialFilterExpression != nil) ||
					(index.Options.PartialFilterExpression != nil && storedIndex.PartialFilterExpression == nil) ||
					!reflect.DeepEqual(index.Options.PartialFilterExpression, storedIndex.PartialFilterExpression) {
					fmt.Printf("PartialFilterExpression of index %s of collection %s modified\r\n", *index.Options.Name, entity)
					modified = true
				}
				if (index.Options.Unique == nil && storedIndex.Unique) || (index.Options.Unique != nil && *index.Options.Unique != storedIndex.Unique) {
					fmt.Printf("Uniqueness of index %s of collection %s modified\r\n", *index.Options.Name, entity)
					modified = true
				}
				if (index.Options.ExpireAfterSeconds == nil && storedIndex.ExpireAfterSeconds > 0) || (index.Options.ExpireAfterSeconds != nil && *index.Options.ExpireAfterSeconds != storedIndex.ExpireAfterSeconds) {
					fmt.Printf("ExpireAfterSeconds of index %s of collection %s modified\r\n", *index.Options.Name, entity)
					modified = true
				}
				if (index.Options.Collation == nil && storedIndex.Collation != nil) ||
					(index.Options.Collation != nil && storedIndex.Collation == nil) ||
					!reflect.DeepEqual(index.Options.Collation, storedIndex.Collation) {
					fmt.Printf("Collation of index %s of collection %s modified\r\n", *index.Options.Name, entity)
					modified = true
				}
				if !modified {
					fmt.Printf("Index %s of collection %s not modified\r\n", *index.Options.Name, entity)
					continue l
				}

				fmt.Printf("Recreate index %s on collection %s\r\n", *index.Options.Name, entity)
				_, err = c.Indexes().DropOne(ctx, storedIndex.Name)
				if err != nil {
					fmt.Printf("Unable to drop index %s of collection %s: %s\r\n", *index.Options.Name, entity, err)
					continue l
				}

				_, err = c.Indexes().CreateOne(ctx, index)
				if err != nil {
					fmt.Printf("Unable to create index %s of collection %s: %s\r\n", *index.Options.Name, entity, err)
					continue l
				}
			}
		}

		// Check for deleted index
	l3:
		for _, storedIndex := range storedIndexes {
			if storedIndex.Name == "_id_" {
				continue l3
			}
			for _, index := range indexes {
				if *index.Options.Name == storedIndex.Name {
					continue l3
				}
			}
			fmt.Printf("Detected deleted index %s on collection %s\r\n", storedIndex.Name, entity)
			_, err = c.Indexes().DropOne(ctx, storedIndex.Name)
			if err != nil {
				fmt.Printf("Unable to drop index %s of collection %s: %s\r\n", storedIndex.Name, entity, err)
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
			fmt.Printf("Create new index %s on collection %s\r\n", *index.Options.Name, entity)
			_, err = c.Indexes().CreateOne(ctx, index)
			if err != nil {
				fmt.Printf("Unable to create index %s of collection %s: %s\r\n", *index.Options.Name, entity, err)
			}
		}

	}
	return nil
}
