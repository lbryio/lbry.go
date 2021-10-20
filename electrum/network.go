package electrum

// copied from https://github.com/d4l3k/go-electrum

import (
	"crypto/tls"
	"encoding/json"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/lbryio/lbry.go/v3/extras/stop"

	"github.com/cockroachdb/errors"
	log "github.com/sirupsen/logrus"
	"go.uber.org/atomic"
)

const (
	ClientVersion   = "0.0.1"
	ProtocolVersion = "1.0"
)

var (
	ErrNotImplemented = errors.New("not implemented")
	ErrNodeConnected  = errors.New("node already connected")
	ErrConnectFailed  = errors.New("failed to connect")
	ErrTimeout        = errors.New("timeout")
)

type response struct {
	data []byte
	err  error
}

type Node struct {
	transport *TCPTransport
	nextId    atomic.Uint32
	grp       *stop.Group

	handlersMu *sync.RWMutex
	handlers   map[uint32]chan response

	pushHandlersMu *sync.RWMutex
	pushHandlers   map[string][]chan response

	timeout time.Duration
}

// NewNode creates a new node.
func NewNode() *Node {
	return &Node{
		handlers:       make(map[uint32]chan response),
		pushHandlers:   make(map[string][]chan response),
		handlersMu:     &sync.RWMutex{},
		pushHandlersMu: &sync.RWMutex{},
		grp:            stop.New(),
		timeout:        1 * time.Second,
	}
}

// Connect creates a new connection to the specified address.
func (n *Node) Connect(addrs []string, config *tls.Config) error {
	if n.transport != nil {
		return errors.WithStack(ErrNodeConnected)
	}

	// shuffle addresses for load balancing
	rand.Shuffle(len(addrs), func(i, j int) { addrs[i], addrs[j] = addrs[j], addrs[i] })

	var err error

	for _, addr := range addrs {
		n.transport, err = NewTransport(addr, config)
		if err == nil {
			break
		}
		if errors.Is(err, ErrTimeout) {
			continue
		}
		if e, ok := err.(*net.OpError); ok && e.Err.Error() == "no such host" {
			// net.errNoSuchHost is not exported, so we have to string-match
			continue
		}
		return errors.WithStack(err)
	}

	if n.transport == nil {
		return errors.WithStack(ErrConnectFailed)
	}

	log.Debugf("wallet connected to %s", n.transport.conn.RemoteAddr())

	n.grp.Add(1)
	go func() {
		defer n.grp.Done()
		<-n.grp.Ch()
		n.transport.Shutdown()
	}()

	n.grp.Add(1)
	go func() {
		defer n.grp.Done()
		n.handleErrors()
	}()

	n.grp.Add(1)
	go func() {
		defer n.grp.Done()
		n.listen()
	}()

	return nil
}

func (n *Node) Shutdown() {
	var addr net.Addr
	if n.transport != nil {
		addr = n.transport.conn.RemoteAddr()
	}
	log.Debugf("shutting down wallet %s", addr)
	n.grp.StopAndWait()
	log.Debugf("wallet stopped")
}

func (n *Node) handleErrors() {
	for {
		select {
		case <-n.grp.Ch():
			return
		case err := <-n.transport.Errors():
			n.err(errors.WithStack(err))
		}
	}
}

// err handles errors produced by the foreign node.
func (n *Node) err(err error) {
	// TODO: Better error handling.
	log.Error(errors.WithStack(err))
}

// listen processes messages from the server.
func (n *Node) listen() {
	for {
		select {
		case <-n.grp.Ch():
			return
		default:
		}

		select {
		case <-n.grp.Ch():
			return
		case bytes := <-n.transport.Responses():
			msg := &struct {
				Id     uint32 `json:"id"`
				Method string `json:"method"`
				Error  struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}{}
			msg2 := &struct {
				Id     uint32 `json:"id"`
				Method string `json:"method"`
				Error  struct {
					Code    int `json:"code"`
					Message struct {
						Code    int    `json:"code"`
						Message string `json:"message"`
					} `json:"message"`
				} `json:"error"`
			}{}
			r := response{}

			err := json.Unmarshal(bytes, msg)
			if err != nil {
				// try msg2, a hack around the weird error-in-error response we sometimes get from wallet server
				// maybe that happens because the wallet server passes a lbrycrd error through to us?
				if err2 := json.Unmarshal(bytes, msg2); err2 == nil {
					err = nil
					msg.Id = msg2.Id
					msg.Method = msg2.Method
					msg.Error = msg2.Error.Message
				}
			}

			if err != nil {
				r.err = errors.WithStack(err)
				n.err(r.err)
			} else if len(msg.Error.Message) > 0 {
				r.err = errors.WithStack(errors.Newf("%d: %s", msg.Error.Code, msg.Error.Message))
			} else {
				r.data = bytes
			}

			if len(msg.Method) > 0 {
				n.pushHandlersMu.RLock()
				handlers := n.pushHandlers[msg.Method]
				n.pushHandlersMu.RUnlock()

				for _, handler := range handlers {
					select {
					case handler <- r:
					default:
					}
				}
			}

			n.handlersMu.RLock()
			c, ok := n.handlers[msg.Id]
			n.handlersMu.RUnlock()
			if ok {
				c <- r
			}
		}
	}
}

// listenPush returns a channel of messages matching the method.
//func (n *Node) listenPush(method string) <-chan []byte {
//	c := make(chan []byte, 1)
//	n.pushHandlersMu.Lock()
//	defer n.pushHandlersMu.Unlock()
//	n.pushHandlers[method] = append(n.pushHandlers[method], c)
//	return c
//}

// request makes a request to the server and unmarshals the response into v.
func (n *Node) request(method string, params []string, v interface{}) error {
	msg := struct {
		Id     uint32   `json:"id"`
		Method string   `json:"method"`
		Params []string `json:"params"`
	}{
		Id:     n.nextId.Load(),
		Method: method,
		Params: params,
	}
	n.nextId.Inc()

	bytes, err := json.Marshal(msg)
	if err != nil {
		return errors.WithStack(err)
	}
	bytes = append(bytes, delimiter)

	c := make(chan response, 1)

	n.handlersMu.Lock()
	n.handlers[msg.Id] = c
	n.handlersMu.Unlock()

	err = n.transport.Send(bytes)
	if err != nil {
		return errors.WithStack(err)
	}

	var r response
	select {
	case <-n.grp.Ch():
		return nil
	case r = <-c:
	case <-time.After(n.timeout):
		r = response{err: ErrTimeout}
	}

	n.handlersMu.Lock()
	delete(n.handlers, msg.Id)
	n.handlersMu.Unlock()

	if r.err != nil {
		return errors.WithStack(r.err)
	}

	return errors.WithStack(json.Unmarshal(r.data, v))
}
