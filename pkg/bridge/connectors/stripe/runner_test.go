package stripe

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestStopTailing(t *testing.T) {

	mock := NewClientMock(t, true)
	tl := NewTimeline(mock, TimelineConfig{
		PageSize: 2,
	}, TimelineState{
		OldestID:     "tx1",
		MoreRecentID: "tx2",
	})

	logger := sharedlogging.GetLogger(context.Background())
	trigger := NewTimelineTrigger(logger, NoOpIngester, tl)
	r := NewRunner(logger, trigger, time.Second)
	go r.Run(context.Background())
	defer r.Stop(context.Background())

	require.False(t, tl.state.NoMoreHistory)

	mock.Expect().RespondsWith(false) // Fetch head
	mock.Expect().RespondsWith(false) // Fetch tail

	require.Eventually(t, func() bool {
		return tl.state.NoMoreHistory
	}, time.Second, 10*time.Millisecond)

}
