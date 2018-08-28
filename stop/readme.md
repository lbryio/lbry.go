# Stop Group

A stop group is meant to help cleanly shut down a component that uses multiple goroutines. 


## The Problem

A shutdown typically works as follows:

- a component receives a shutdown signal
- the component passes the shutdown signal to each outstanding goroutine
- each goroutine stops whatever it is doing and returns
- once all the goroutines have returned, the component does any final cleanup and signals that it is done

There are several gotchas in this process:

- each goroutine must be tracked, so all can be stopped
- the component should not expose implementation details about how many goroutines it uses and how they are managed
- goroutines may be blocked by a read on a channel. they need to be unblocked during shutdown
- goroutines may take a while to finish, and may finish in any order. component shutdown is not complete until all goroutines finish
- using a channel to send the shutdown signal is complicated (doing things in the correct order, closing an already-closed channel, etc)


## The Solution

The solution is a stop group. A stop group is a combination of a `sync.WaitGroup` and a cancelable `context`. Here's how it works:

```
grp := stop.New()
action := func() { ... }
```

All goroutines are started in the start group.

```
grp.Add(1)
go func() {
  defer grp.Done()
  action()
}
```


Any goroutine that may be blocked by a channel read has a simple way of unblocking on shutdown

```
for {
  select {
  case text := <-actionCh:
    fmt.Printf("Got some text: %s", text)
  case <-grp.Ch():
    return
  }
}
```


Shutting down synchronously is easy

```
grp.StopAndWait()
```


## Example

```
type Server struct {
	grp   *stop.Group
}

func NewServer() *Server {
	return &Server{
		grp:     stop.New(),
	}
}

func (s *Server) Shutdown() {
	s.grp.StopAndWait()
}

func (s *Server) Start(address string) error {
	l, err := net.Listen(network, address)
	if err != nil {
		return err
	}
	log.Println("listening on " + address)

	s.grp.Add(1)
	go func() {
		defer s.grp.Done()
		<-s.grp.Ch()
		err := l.Close()
		if err != nil {
			log.Errorln(err)
		}
	}()

	s.grp.Add(1)
	go func() {
		defer s.grp.Done()
		for {
			select {
			case <-s.grp.Ch():
				return
			case <-time.Tick(10 * time.Second):
				log.Println("still running")
			}
		}
	}()

	s.grp.Add(1)
	go func() {
		defer s.grp.Done()
		s.listenAndServe(l)
	}()

	return nil
}

```
