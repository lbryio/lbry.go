package stopOnce

import "sync"

type Stopper struct {
	sync.WaitGroup
	ch   chan struct{}
	once sync.Once
}

func New() *Stopper {
	s := &Stopper{}
	s.ch = make(chan struct{})
	s.once = sync.Once{}
	return s
}

func (s *Stopper) Ch() <-chan struct{} {
	return s.ch
}

func (s *Stopper) Stop() {
	s.once.Do(func() {
		close(s.ch)
	})
}

func (s *Stopper) StopAndWait() {
	s.Stop()
	s.Wait()
}
