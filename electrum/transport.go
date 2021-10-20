package electrum

// copied from https://github.com/d4l3k/go-electrum

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"net"
	"time"

	"github.com/lbryio/lbry.go/v3/extras/stop"

	"github.com/cockroachdb/errors"
	log "github.com/sirupsen/logrus"
)

type TCPTransport struct {
	conn      net.Conn
	responses chan []byte
	errors    chan error
	grp       *stop.Group
}

func NewTransport(addr string, config *tls.Config) (*TCPTransport, error) {
	var conn net.Conn
	var err error

	timeout := 5 * time.Second
	if config != nil {
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: timeout}, "tcp", addr, config)
	} else {
		conn, err = net.DialTimeout("tcp", addr, timeout)
	}
	if err != nil {
		return nil, err
	}

	t := &TCPTransport{
		conn:      conn,
		responses: make(chan []byte),
		errors:    make(chan error),
		grp:       stop.New(),
	}

	t.grp.Add(1)
	go func() {
		defer t.grp.Done()
		<-t.grp.Ch()
		t.close()
	}()

	t.grp.Add(1)
	go func() {
		defer t.grp.Done()
		t.listen()
	}()

	err = t.test()
	if err != nil {
		t.grp.StopAndWait()
		return nil, errors.WithMessage(err, addr)
	}

	return t, nil
}

const delimiter = byte('\n')

func (t *TCPTransport) Send(body []byte) error {
	log.Debugf("%s <- %s", t.conn.RemoteAddr(), body)
	_, err := t.conn.Write(body)
	return err
}

func (t *TCPTransport) Responses() <-chan []byte { return t.responses }
func (t *TCPTransport) Errors() <-chan error     { return t.errors }
func (t *TCPTransport) Shutdown()                { t.grp.StopAndWait() }

func (t *TCPTransport) listen() {
	reader := bufio.NewReader(t.conn)
	for {
		line, err := reader.ReadBytes(delimiter)
		if err != nil {
			t.error(err)
			return
		}

		log.Debugf("%s -> %s", t.conn.RemoteAddr(), line)

		t.responses <- line
	}
}

func (t *TCPTransport) error(err error) {
	select {
	case t.errors <- err:
	default:
	}
}

func (t *TCPTransport) test() error {
	err := t.Send([]byte(`{"id":1,"method":"server.version"}` + "\n"))
	if err != nil {
		return errors.WithStack(err)
	}

	var data []byte
	select {
	case data = <-t.Responses():
	case <-time.Tick(1 * time.Second):
		return errors.WithStack(ErrTimeout)
	}

	var response struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	err = json.Unmarshal(data, &response)
	if err != nil {
		return errors.WithStack(err)
	}
	if response.Error.Message != "" {
		return errors.WithStack(errors.New(response.Error.Message))
	}
	return nil
}

func (t *TCPTransport) close() {
	err := t.conn.Close()
	if err != nil {
		t.error(err)
	}
}
