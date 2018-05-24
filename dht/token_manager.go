package dht

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/lbryio/lbry.go/stopOnce"
)

type tokenManager struct {
	secret     []byte
	prevSecret []byte
	lock       *sync.RWMutex
	wg         *sync.WaitGroup
	done       *stopOnce.Stopper
}

func (tm *tokenManager) Start(interval time.Duration) {
	tm.secret = make([]byte, 64)
	tm.prevSecret = make([]byte, 64)
	tm.lock = &sync.RWMutex{}
	tm.wg = &sync.WaitGroup{}
	tm.done = stopOnce.New()

	tm.rotateSecret()

	tm.wg.Add(1)
	go func() {
		defer tm.wg.Done()
		tick := time.NewTicker(interval)
		for {
			select {
			case <-tick.C:
				tm.rotateSecret()
			case <-tm.done.Ch():
				return
			}
		}
	}()
}

func (tm *tokenManager) Stop() {
	tm.done.Stop()
	tm.wg.Wait()
}

func (tm *tokenManager) Get(nodeID Bitmap, addr *net.UDPAddr) string {
	return genToken(tm.secret, nodeID, addr)
}

func (tm *tokenManager) Verify(token string, nodeID Bitmap, addr *net.UDPAddr) bool {
	return token == genToken(tm.secret, nodeID, addr) || token == genToken(tm.prevSecret, nodeID, addr)
}

func genToken(secret []byte, nodeID Bitmap, addr *net.UDPAddr) string {
	buf := bytes.Buffer{}
	buf.Write(nodeID[:])
	buf.Write(addr.IP)
	buf.WriteString(strconv.Itoa(addr.Port))
	buf.Write(secret)
	t := sha256.Sum256(buf.Bytes())
	return string(t[:])
}

func (tm *tokenManager) rotateSecret() {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	copy(tm.prevSecret, tm.secret)

	_, err := rand.Read(tm.secret)
	if err != nil {
		panic(err)
	}
}
