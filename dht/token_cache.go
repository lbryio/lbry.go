package dht

import (
	"sync"
	"time"

	"github.com/lbryio/reflector.go/dht/bits"

	"github.com/lbryio/lbry.go/stopOnce"
)

// TODO: this should be moved out of dht and into node, and it should be completely hidden inside node. dht should not need to know about tokens

type tokenCacheEntry struct {
	token      string
	receivedAt time.Time
}

type tokenCache struct {
	node       *Node
	tokens     map[string]tokenCacheEntry
	expiration time.Duration
	lock       *sync.RWMutex
}

func newTokenCache(node *Node, expiration time.Duration) *tokenCache {
	tc := &tokenCache{}
	tc.node = node
	tc.tokens = make(map[string]tokenCacheEntry)
	tc.expiration = expiration
	tc.lock = &sync.RWMutex{}
	return tc
}

func (tc *tokenCache) Get(c Contact, hash bits.Bitmap, cancelCh stopOnce.Chan) string {
	tc.lock.RLock()
	token, exists := tc.tokens[c.String()]
	tc.lock.RUnlock()

	if exists && time.Since(token.receivedAt) < tc.expiration {
		return token.token
	}

	resCh := tc.node.SendAsync(c, Request{
		Method: findValueMethod,
		Arg:    &hash,
	})

	var res *Response

	select {
	case res = <-resCh:
	case <-cancelCh:
		return ""
	}

	if res == nil {
		return ""
	}

	tc.lock.Lock()
	tc.tokens[c.String()] = tokenCacheEntry{
		token:      res.Token,
		receivedAt: time.Now(),
	}
	tc.lock.Unlock()

	return res.Token
}
