/*Package stop implements the stopper pattern for golang concurrency management. The main use case is to gracefully
 exit an application. The pattern allows for a hierarchy of stoppers. Each package should have its own unexported stopper.
 The package should maintain startup and shutdown exported methods. If the stopper should stop when another stopper for
 a different package stops, the parent argument for the initialization of a stopper be used to create the dependency.

The package also comes with a debugging tool to help in determining why and where a stopper is not stopping as expected.
If a more complex concurrency is used, it is recommended to implement the library using `DoneNamed` and `AddNamed`.
In addition to the standard `Done` functionality, it allows a functional named representation of the type of go routine
being completed. This works in conjunction with `AddNamed`. If the init of the stopper Group happens with `NewDebug`
instead of `New` special tracking and logging is enabled where it will print out the remaining routines' functional name
and how many it is waiting on to complete the stop after each call to `DoneNamed`. This allows easy debugging by just
changing the init of the stopper instead of all the references as long as the library is implemented with `DoneNamed`
and `AddNamed`. For simple uses of the stopper pattern this is not needed and the standard `Add` and `Done` should be used.
*/
package stop

import (
	"context"
	"log"
	"sync"
)

// Chan is a receive-only channel
type Chan <-chan struct{}

// Group extends sync.WaitGroup to add a convenient way to stop running goroutines
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
	s.mu = &sync.Mutex{}

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

//AddNamed is the same as Add but will register the functional name of the routine for later output. See `DoneNamed`.
func (s *Group) AddNamed(delta int, name string) {
	s.Add(delta)

	if s.waitingOn != nil {
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
}

//DoneNamed is the same as `Done` but will output the functional name of all remaining named routines and the waiting on count.
func (s *Group) DoneNamed(name string) {
	defer s.Done()

	if s.waitingOn != nil {
		s.mu.Lock()
		if current, ok := s.waitingOn[name]; ok {
			if current <= 1 {
				delete(s.waitingOn, name)
			} else {
				s.waitingOn[name] = current - 1
			}
		} else {
			log.Printf("%s is not recorded in stop group map", name)
		}
		s.mu.Unlock()
		s.listWaitingOn()
	}
}

func (s *Group) listWaitingOn() {
	if s.waitingOn != nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		log.Printf("-->> LIST ROUTINES WAITING ON")
		for k, v := range s.waitingOn {
			if v > 0 {
				log.Printf("waiting on %d %s routines...", v, k)
			}
		}
	}
}
