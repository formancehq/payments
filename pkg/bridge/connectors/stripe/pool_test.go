package stripe

import (
	"context"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
)

func TestPoolCancel(t *testing.T) {
	poolSize := 3
	p := NewPool(poolSize)
	go p.Run(context.Background())

	wg := sync.WaitGroup{}
	wg.Add(poolSize)
	for i := 0; i < poolSize; i++ {
		go func() {
			err := p.Push(context.Background(), func(ctx context.Context) error {
				wg.Done()
				select {
				case <-ctx.Done():
					return nil
				}
			})
			require.NoError(t, err)
		}()
	}
	wg.Wait()

	p.Stop(context.Background())

}
