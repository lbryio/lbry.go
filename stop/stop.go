package stop

import (
	"context"
	"log"
	"sync"
)

// Chan is a receive-only channel
type Chan <-chan struct{}

// Stopper extends sync.WaitGroup to add a convenient way to stop running goroutines
type Group struct {
	sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc

	mu        *sync.Mutex
	waitingOn map[string]int
}
type Stopper = Group

// New allocates and returns a new instance. Use New(parent) to create an instance that is stopped when parent is stopped.
func New(parent ...*Group) *Group {
	s := &Group{mu: &sync.Mutex{}}
	ctx := context.Background()
	if len(parent) > 0 && parent[0] != nil {
		ctx = parent[0].ctx
	}
	s.ctx, s.cancel = context.WithCancel(ctx)
	return s
}

// Ch returns a channel that will be closed when Stop is called.
func (s *Group) Ch() Chan {
	return s.ctx.Done()
}

// Stop signals any listening processes to stop. After the first call, Stop() does nothing.
func (s *Group) Stop() {
	s.cancel()
}

// StopAndWait is a convenience method to close the channel and wait for goroutines to return.
func (s *Group) StopAndWait() {
	s.Stop()
	s.Wait()
}

// Child returns a new instance that will be stopped when s is stopped.
func (s *Group) Child() *Group {
	return New(s)
}

func (s *Group) DebugAdd(delta int, name string) {
	s.Add(delta)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.waitingOn == nil {
		s.waitingOn = make(map[string]int)
	}

	if current, ok := s.waitingOn[name]; ok {
		s.waitingOn[name] = current + 1
	} else {
		s.waitingOn[name] = 1
	}
}

func (s *Group) DebugDone(name string) {
	defer s.Done()

	s.mu.Lock()
	defer s.mu.Unlock()

	if current, ok := s.waitingOn[name]; ok {
		if current <= 1 {
			delete(s.waitingOn, name)
		} else {
			s.waitingOn[name] = current - 1
		}
	} else {
		log.Printf("%s is not recorded in stop group map", name)
	}

	log.Printf("-->> LIST WAITING ON")

	for k, v := range s.waitingOn {
		if v > 0 {
			log.Printf("waiting on %d %s routines...", v, k)
		}
	}
}
