package stripe

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	payments "github.com/numary/payments/pkg"
	"github.com/numary/payments/pkg/bridge"
	. "github.com/numary/payments/pkg/paymentstesting"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v72"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"testing"
)

func TestIngester(t *testing.T) {
	RunWithMock(t, func(t *mtest.T) {
		connectorName := "testing"
		logger := sharedlogging.NewNoOpLogger()
		bridgeIngester := bridge.NewDefaultIngester[State](connectorName, t.DB, logger, nil)
		logObjectStorage := bridge.NewDefaultLogObjectStorage(connectorName, t.DB, logger)

		mainIngester := NewDefaultIngester(connectorName, "", sharedlogging.NewNoOpLogger(), bridgeIngester, logObjectStorage)
		accountIngester := NewDefaultIngester(connectorName, "account1", sharedlogging.NewNoOpLogger(), bridgeIngester, logObjectStorage)

		mainState := TimelineState{
			OldestID:     "oldest-id",
			MoreRecentID: "more-recent-id",
		}
		err := mainIngester.Ingest(context.Background(), []stripe.BalanceTransaction{}, mainState, false)
		require.NoError(t, err)

		account1State := TimelineState{
			OldestID:     "oldest-id-2",
			MoreRecentID: "more-recent-id-2",
		}
		err = accountIngester.Ingest(context.Background(), []stripe.BalanceTransaction{}, account1State, false)
		require.NoError(t, err)

		ret := t.DB.Collection(payments.ConnectorStatesCollection).FindOne(context.Background(), map[string]interface{}{
			"provider": connectorName,
		})
		require.NoError(t, ret.Err())
		state := State{}
		require.NoError(t, ret.Decode(&state))
		require.Equal(t, State{
			TimelineState: mainState,
			Accounts: map[string]TimelineState{
				"account1": account1State,
			},
		}, state)
	})
}
