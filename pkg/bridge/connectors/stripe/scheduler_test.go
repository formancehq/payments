package stripe

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v72"
	"testing"
	"time"
)

func TestScheduleNewAccounts(t *testing.T) {
	mock := NewClientMock(t)
	mock.Expect().RespondsWith(false, &stripe.BalanceTransaction{
		ID: "tx1",
		Source: &stripe.BalanceTransactionSource{
			Transfer: &stripe.Transfer{
				Destination: &stripe.TransferDestination{
					ID: "connected-account-1",
				},
			},
			Type: stripe.BalanceTransactionSourceTypeTransfer,
		},
		Type: "transfer",
	})

	scheduler := NewScheduler(bridge.NoOpLogObjectStorage, sharedlogging.GetLogger(context.Background()), bridge.NoOpIngester[State](), mock, Config{
		Pool:          1,
		PollingPeriod: time.Second,
		TimelineConfig: TimelineConfig{
			PageSize: 2,
		},
	}, State{
		TimelineState: TimelineState{
			OldestID:     "tx1",
			MoreRecentID: "tx1",
		},
		Accounts: map[string]TimelineState{},
	})
	go scheduler.Start(context.Background())
	defer scheduler.Stop(context.Background())

	require.Eventually(t, func() bool {
		return len(scheduler.accountRunners) == 1
	}, 3*time.Second, 10*time.Millisecond)
}
