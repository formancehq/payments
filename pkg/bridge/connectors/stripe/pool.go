package stripe

import (
	"context"
	"fmt"
)

type poolFunction func(ctx context.Context) error

type poolAction struct {
	fn  poolFunction
	ret chan error
}

type pool struct {
	actions  chan *poolAction
	workers  chan struct{}
	stopChan chan chan error
}

func (p *pool) Push(ctx context.Context, fn func(ctx context.Context) error) error {
	action := &poolAction{
		fn:  fn,
		ret: make(chan error, 1),
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case p.actions <- action:
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-action.ret:
			return err
		}
	}
}

func (p *pool) Stop(ctx context.Context) error {
	ch := make(chan error)
	p.stopChan <- ch
	return <-ch
}

func (p *pool) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case errCh := <-p.stopChan:
			close(p.actions)
			for i := 0; i < cap(p.workers); i++ { // Drain all workers
				<-p.workers
			}
			close(p.workers)

			for action := range p.actions { // Drain all pending actions
				action.ret <- fmt.Errorf("pool stopped")
				close(action.ret)
			}
			errCh <- nil
			return nil
		case action := <-p.actions:
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-p.workers:
				go func(action *poolAction) {
					defer func() {
						close(action.ret)
						select {
						case <-ctx.Done():
						case p.workers <- struct{}{}:
						}
					}()
					action.fn(ctx)
				}(action)
			}
		}
	}
}

func NewPool(size int) *pool {
	p := &pool{
		actions:  make(chan *poolAction),
		workers:  make(chan struct{}, size),
		stopChan: make(chan chan error),
	}
	for i := 0; i < size; i++ {
		p.workers <- struct{}{}
	}
	return p
}
