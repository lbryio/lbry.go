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
	waitingOn map[string]int
	l         sync.RWMutex
	sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}
type Stopper = Group

// New allocates and returns a new instance. Use New(parent) to create an instance that is stopped when parent is stopped.
func New(parent ...*Group) *Group {
	s := &Group{}
	ctx := context.Background()
	if len(parent) > 0 && parent[0] != nil {
		ctx = parent[0].ctx
	}
	s.ctx, s.cancel = context.WithCancel(ctx)
	return s
}

// NewDebug allows you to debug the go routines the group waits on. In order to leverage this, AddNamed and DoneNamed should be used.
func NewDebug(parent ...*Group) *Group {
	s := New(parent...)
	s.waitingOn = make(map[string]int)
	s.l = sync.RWMutex{}

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

func (s *Group) AddNamed(delta int, name string) {
	if s.waitingOn != nil {
		s.l.Lock()
		defer s.l.Unlock()
		_, ok := s.waitingOn[name]
		if !ok {
			s.waitingOn[name] = 1
		} else {
			s.waitingOn[name] = s.waitingOn[name] + 1
		}
	}
	s.Add(delta)

}

func (s *Group) DoneNamed(name string) {
	if s.waitingOn != nil {
		s.l.Lock()
		defer s.l.Unlock()
		_, ok := s.waitingOn[name]
		if !ok {
			log.Printf("%s is not recorded in stop group map")
		} else {
			s.waitingOn[name] = s.waitingOn[name] - 1
		}
		log.Printf("-->> LIST WAITING ON")
		for k, v := range s.waitingOn {
			if v > 0 {
				log.Printf("waiting on %d %s routines...", v, k)
			}
		}
	}
	s.Done()
}
