package stripe

import (
	"context"
	"fmt"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v72"
	"testing"
	"time"
)

func TestTimelineTrigger(t *testing.T) {
	const txCount = 12

	mock := NewClientMock(t)
	ref := time.Now().Add(-time.Minute * time.Duration(txCount) / 2)
	tl := NewTimeline(mock, TimelineConfig{
		PageSize: 2,
	}, TimelineState{}, WithStartingAt(ref))

	ingestedTx := make([]*stripe.BalanceTransaction, 0)
	trigger := NewTimelineTrigger(
		sharedlogging.GetLogger(context.Background()),
		IngesterFn(func(ctx context.Context, batch []*stripe.BalanceTransaction, commitState TimelineState, tail bool) error {
			ingestedTx = append(ingestedTx, batch...)
			return nil
		}),
		tl,
	)

	allTxs := make([]*stripe.BalanceTransaction, txCount)
	for i := 0; i < txCount/2; i++ {
		allTxs[txCount/2+i] = &stripe.BalanceTransaction{
			ID:      fmt.Sprintf("%d", txCount/2+i),
			Created: ref.Add(-time.Duration(i) * time.Minute).Unix(),
		}
		allTxs[txCount/2-i-1] = &stripe.BalanceTransaction{
			ID:      fmt.Sprintf("%d", txCount/2-i-1),
			Created: ref.Add(time.Duration(i) * time.Minute).Unix(),
		}
	}

	for i := 0; i < txCount/2; i += 2 {
		mock.Expect().Limit(2).RespondsWith(i < txCount/2-2, allTxs[txCount/2+i], allTxs[txCount/2+i+1])
	}
	for i := 0; i < txCount/2; i += 2 {
		mock.Expect().Limit(2).RespondsWith(i < txCount/2-2, allTxs[txCount/2-i-2], allTxs[txCount/2-i-1])
	}

	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(time.Second))

	trigger.Fetch(ctx)
	require.Len(t, ingestedTx, txCount)

}
