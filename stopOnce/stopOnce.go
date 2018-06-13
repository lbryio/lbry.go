package stopOnce

import "sync"

// Chan is a receive-only channel
type Chan <-chan struct{}

// Stopper extends sync.WaitGroup to add a convenient way to stop running goroutines
type Stopper struct {
	sync.WaitGroup
	ch   chan struct{}
	once sync.Once
}

// New allocates and returns a new Stopper instance
func New() *Stopper {
	s := &Stopper{}
	s.ch = make(chan struct{})
	s.once = sync.Once{}
	return s
}

// Ch returns a channel that will be closed when Stop is called
func (s *Stopper) Ch() Chan {
	return s.ch
}

// Stop closes the stopper channel. It is safe to call Stop many times. The channel will only be closed the first time.
func (s *Stopper) Stop() {
	s.once.Do(func() {
		close(s.ch)
	})
}

// StopAndWait is a convenience method to close the channel and wait for goroutines to return
func (s *Stopper) StopAndWait() {
	s.Stop()
	s.Wait()
}

// Link will stop s if upstream is stopped.
// If you use Link, make sure you stop `s` when you're done with it. Otherwise this goroutine will be leaked.
func (s *Stopper) Link(upstream Chan) {
	go func() {
		select {
		case <-upstream: // linked Stopper is stopped
			s.Stop()
		case <-s.Ch(): // this Stopper is stopped
		}
	}()
}
