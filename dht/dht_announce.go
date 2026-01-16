package dht

import (
	"container/ring"
	"context"
	"math"
	"sync"
	"time"

	"github.com/lbryio/lbry.go/v3/dht/bits"

	"github.com/cockroachdb/errors"
	"golang.org/x/time/rate"
)

type queueEdit struct {
	hash bits.Bitmap
	add  bool
}

const (
	announceStarted = "started"
	announceFinishd = "finished"
)

type announceNotification struct {
	hash   bits.Bitmap
	action string
	err    error
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

	var queue *ring.Ring
	hashes := make(map[bits.Bitmap]*ring.Ring)

	var announceNextHash <-chan time.Time
	timer := time.NewTimer(math.MaxInt64)
	timer.Stop()

	limitCh := make(chan time.Time)
	dht.grp.Add(1)
	go func() {
		defer dht.grp.Done()
		limiter := rate.NewLimiter(rate.Limit(dht.conf.AnnounceRate), dht.conf.AnnounceRate)
		for {
			err := limiter.Wait(context.Background()) // TODO: should use grp.ctx somehow? so when grp is closed, wait returns
			if err != nil {
				log.Error(errors.WithMessage(err, "rate limiter"))
				continue
			}
			select {
			case limitCh <- time.Now():
			case <-dht.grp.Ch():
				return
			}
		}
	}()

	maintenance := time.NewTicker(1 * time.Minute)

	// TODO: work to space hash announces out so they aren't bunched up around the reannounce time. track time since last announce. if its been more than the ideal time (reannounce time / numhashes), start announcing hashes early

	for {
		select {
		case <-dht.grp.Ch():
			return

		case <-maintenance.C:
			maxAnnounce := dht.conf.AnnounceRate * int(dht.conf.ReannounceTime.Seconds())
			if len(hashes) > maxAnnounce {
				// TODO: send this to slack
				log.Warnf("DHT has %d hashes, but can only announce %d hashes in the %s reannounce window. Raise the announce rate or spawn more nodes.",
					len(hashes), maxAnnounce, dht.conf.ReannounceTime.String())
			}

		case change := <-dht.announceAddRemove:
			if change.add {
				if _, exists := hashes[change.hash]; exists {
					continue
				}

				r := ring.New(1)
				r.Value = hashAndTime{hash: change.hash}
				if queue != nil {
					queue.Prev().Link(r)
				}
				queue = r
				hashes[change.hash] = r
				announceNextHash = limitCh // announce next hash ASAP
			} else {
				r, exists := hashes[change.hash]
				if !exists {
					continue
				}

				delete(hashes, change.hash)

				if len(hashes) == 0 {
					queue = ring.New(0)
					announceNextHash = nil // no hashes to announce, wait indefinitely
				} else {
					if r == queue {
						queue = queue.Next() // don't lose our pointer
					}
					r.Prev().Link(r.Next())
				}
			}

		case <-announceNextHash:
			dht.grp.Add(1)
			ht := queue.Value.(hashAndTime)

			if !ht.lastAnnounce.IsZero() {
				nextAnnounce := ht.lastAnnounce.Add(dht.conf.ReannounceTime)
				if nextAnnounce.After(time.Now()) {
					timer.Reset(time.Until(nextAnnounce))
					announceNextHash = timer.C // wait until next hash should be announced
					continue
				}
			}

			if dht.conf.AnnounceNotificationCh != nil {
				dht.conf.AnnounceNotificationCh <- announceNotification{
					hash:   ht.hash,
					action: announceStarted,
				}
			}

			go func(hash bits.Bitmap) {
				defer dht.grp.Done()
				err := dht.announce(hash)
				if err != nil {
					log.Error(errors.WithMessage(err, "announce"))
				}

				if dht.conf.AnnounceNotificationCh != nil {
					dht.conf.AnnounceNotificationCh <- announceNotification{
						hash:   ht.hash,
						action: announceFinishd,
						err:    err,
					}
				}
			}(ht.hash)

			queue.Value = hashAndTime{hash: ht.hash, lastAnnounce: time.Now()}
			queue = queue.Next()
			announceNextHash = limitCh // announce next hash ASAP
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
