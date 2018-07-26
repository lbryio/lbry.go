package dht

import (
	"container/ring"
	"context"
	"sync"
	"time"

	"github.com/lbryio/lbry.go/errors"
	"github.com/lbryio/reflector.go/dht/bits"
	"golang.org/x/time/rate"
)

type queueEdit struct {
	hash bits.Bitmap
	add  bool
}

// Add adds the hash to the list of hashes this node is announcing
func (dht *DHT) Add(hash bits.Bitmap) {
	dht.announceAddRemove <- queueEdit{hash: hash, add: true}
}

// Remove removes the hash from the list of hashes this node is announcing
func (dht *DHT) Remove(hash bits.Bitmap) {
	dht.announceAddRemove <- queueEdit{hash: hash, add: false}
}

func (dht *DHT) runAnnouncer() {
	type hashAndTime struct {
		hash         bits.Bitmap
		lastAnnounce time.Time
	}

	queue := ring.New(0)
	hashes := make(map[bits.Bitmap]*ring.Ring)
	limiter := rate.NewLimiter(rate.Limit(dht.conf.AnnounceRate), dht.conf.AnnounceRate*dht.conf.AnnounceBurst)

	var announceNextHash <-chan time.Time
	timer := time.NewTimer(0)
	closedCh := make(chan time.Time)
	close(closedCh)

	for {
		select {
		case <-dht.grp.Ch():
			return

		case change := <-dht.announceAddRemove:
			if change.add {
				r := ring.New(1)
				r.Value = hashAndTime{hash: change.hash}
				queue.Prev().Link(r)
				queue = r
				hashes[change.hash] = r
				announceNextHash = closedCh // don't wait to announce next hash
			} else {
				if r, exists := hashes[change.hash]; exists {
					delete(hashes, change.hash)
					if len(hashes) == 0 {
						queue = ring.New(0)
						announceNextHash = make(chan time.Time) // no hashes to announce, wait indefinitely
					} else {
						if r == queue {
							queue = queue.Next() // don't lose our pointer
						}
						r.Prev().Link(r.Next())
					}
				}
			}

		case <-announceNextHash:
			limiter.Wait(context.Background()) // TODO: should use grp.ctx somehow
			dht.grp.Add(1)
			ht := queue.Value.(hashAndTime)

			if !ht.lastAnnounce.IsZero() {
				nextAnnounce := ht.lastAnnounce.Add(dht.conf.ReannounceTime)
				if nextAnnounce.Before(time.Now()) {
					timer.Reset(time.Until(nextAnnounce))
					announceNextHash = timer.C // wait until next hash should be announced
					continue
				}
			}

			go func(hash bits.Bitmap) {
				defer dht.grp.Done()
				err := dht.announce(hash)
				if err != nil {
					log.Error(errors.Prefix("announce", err))
				}
			}(ht.hash)

			queue.Value = hashAndTime{hash: ht.hash, lastAnnounce: time.Now()}
			queue = queue.Next()
			announceNextHash = closedCh // don't wait to announce next hash
		}
	}
}

// Announce announces to the DHT that this node has the blob for the given hash
func (dht *DHT) announce(hash bits.Bitmap) error {
	contacts, _, err := FindContacts(dht.node, hash, false, dht.grp.Child())
	if err != nil {
		return err
	}

	// self-store if we found less than K contacts, or we're closer than the farthest contact
	if len(contacts) < bucketSize {
		contacts = append(contacts, dht.contact)
	} else if hash.Closer(dht.node.id, contacts[bucketSize-1].ID) {
		contacts[bucketSize-1] = dht.contact
	}

	wg := &sync.WaitGroup{}
	for _, c := range contacts {
		wg.Add(1)
		go func(c Contact) {
			dht.store(hash, c)
			wg.Done()
		}(c)
	}

	wg.Wait()

	return nil
}

func (dht *DHT) store(hash bits.Bitmap, c Contact) {
	if dht.contact.ID == c.ID {
		// self-store
		c.PeerPort = dht.conf.PeerProtocolPort
		dht.node.Store(hash, c)
		return
	}

	dht.node.SendAsync(c, Request{
		Method: storeMethod,
		StoreArgs: &storeArgs{
			BlobHash: hash,
			Value: storeArgsValue{
				Token:  dht.tokenCache.Get(c, hash, dht.grp.Ch()),
				LbryID: dht.contact.ID,
				Port:   dht.conf.PeerProtocolPort,
			},
		},
	})
}
