package stripe

import (
	"context"
	"errors"
)

var ErrPoolClosed = errors.New("pool closed")

type poolFunction func(ctx context.Context) error

type poolAction struct {
	fn  poolFunction
	ret chan error
}

type pool struct {
	actions  chan *poolAction
	workers  chan struct{}
	stopChan chan chan struct{}
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

func (p *pool) Stop(ctx context.Context) {
	ch := make(chan struct{})
	select {
	case <-ctx.Done():
	case p.stopChan <- ch:
		select {
		case <-ctx.Done():
		case <-ch:
		}
	}
}

func (p *pool) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)

	closeAction := func(action *poolAction, err error) {
		action.ret <- err
		close(action.ret)
	}

	stop := func() {
		close(p.actions) // Avoid new workers
		cancel()         // Cancel active workers

		for i := 0; i < cap(p.workers); i++ { // Drain all workers
			<-p.workers
		}
		close(p.workers) // Close workers

		for action := range p.actions { // Drain all pending actions
			closeAction(action, ErrPoolClosed)
		}
	}

	for {
		select {
		case <-ctx.Done():
			stop()
			return ctx.Err()
		case ch := <-p.stopChan:
			stop()
			close(ch)
			return nil
		case action := <-p.actions:
			select {
			case ch := <-p.stopChan:
				closeAction(action, ErrPoolClosed)
				stop()
				close(ch)
				return nil
			case <-ctx.Done():
				stop()
				return ctx.Err()
			case <-p.workers:
				go func(action *poolAction) {
					// TODO: Catch potential panic on function call
					defer func() {
						p.workers <- struct{}{}
					}()
					closeAction(action, action.fn(ctx))
				}(action)
			}
		}
	}
}

func NewPool(size int) *pool {
	p := &pool{
		actions:  make(chan *poolAction),
		workers:  make(chan struct{}, size),
		stopChan: make(chan chan struct{}),
	}
	for i := 0; i < size; i++ {
		p.workers <- struct{}{}
	}
	return p
}
