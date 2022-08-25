package ingestion

import (
	"context"
	"testing"
	"time"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/core"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

type State struct {
	Counter int
}

func TestIngester(t *testing.T) {
	mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock)).Run("Schedule task", func(mt *mtest.T) {

		provider := "testing"
		ingester := NewDefaultIngester(provider, uuid.New(), mt.DB, sharedlogging.NewNoOpLogger(), nil)

		mt.AddMockResponses(
			bson.D{
				{Key: "ok", Value: 1},
				{Key: "value", Value: bson.D{}},
			}, // Find payment update
			mtest.CreateSuccessResponse(), // Respond to state update
			mtest.CreateSuccessResponse(), // Commit transaction
		)

		err := ingester.Ingest(context.Background(), Batch{
			{
				Referenced: core.Referenced{
					Reference: "p1",
					Type:      core.TypePayIn,
				},
				Payment: &core.Data{
					Status:        core.StatusSucceeded,
					InitialAmount: 100,
					Scheme:        core.SchemeOther,
					Asset:         "USD/2",
					CreatedAt:     time.Now(),
				},
			},
		}, State{
			Counter: 1,
		})
		require.NoError(t, err)
	})
}
