package ingestion

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"testing"
	"time"
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
				{"ok", 1},
				{"value", bson.D{}},
			}, // Find payment update
			mtest.CreateSuccessResponse(), // Respond to state update
			mtest.CreateSuccessResponse(), // Commit transaction
		)

		err := ingester.Ingest(context.Background(), Batch{
			{
				Referenced: payments.Referenced{
					Reference: "p1",
					Type:      payments.TypePayIn,
				},
				Payment: &payments.Data{
					Status:        payments.StatusSucceeded,
					InitialAmount: 100,
					Scheme:        payments.SchemeOther,
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
