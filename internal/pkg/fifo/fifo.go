package fifo

import "sync"

type FIFO[ITEM any] struct {
	mu    sync.Mutex
	items []ITEM
}

func (s *FIFO[ITEM]) Pop() (ret ITEM, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.items) == 0 {
		return
	}
	ret = s.items[0]
	ok = true
	if len(s.items) == 1 {
		s.items = make([]ITEM, 0)
		return
	}
	s.items = s.items[1:]
	return
}

func (s *FIFO[ITEM]) Peek() (ret ITEM, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.items) == 0 {
		return
	}
	return s.items[0], true
}

func (s *FIFO[ITEM]) Push(i ITEM) *FIFO[ITEM] {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items = append(s.items, i)
	return s
}

func (s *FIFO[ITEM]) Empty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.items) == 0
}
