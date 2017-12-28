package stopOnce

import "sync"

type Stopper struct {
	ch   chan struct{}
	once sync.Once
}

func New() *Stopper {
	s := Stopper{}
	s.ch = make(chan struct{})
	return &s
}

func (s Stopper) Chan() <-chan struct{} {
	return s.ch
}

func (s Stopper) Stop() {
	s.once.Do(func() {
		close(s.ch)
	})
}
