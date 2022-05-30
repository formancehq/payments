package stripe

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestStopTailing(t *testing.T) {

	mock := NewClientMock(t)
	tl := NewTimeline(mock, TimelineConfig{
		PageSize: 2,
	}, TimelineState{
		OldestID:     "tx1",
		MoreRecentID: "tx2",
	})

	r := NewRunner(sharedlogging.GetLogger(context.Background()), NoOpIngester, tl, time.Second)
	go r.Run(context.Background())
	defer r.Stop(context.Background())

	require.True(t, r.IsTailing())

	mock.Expect().RespondsWith(false) // Fetch head
	mock.Expect().RespondsWith(false) // Fetch tail

	require.Eventually(t, func() bool {
		return !r.IsTailing()
	}, time.Second, 10*time.Millisecond)

}
