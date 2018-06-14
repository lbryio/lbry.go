package dht

import (
	"sort"
	"sync"
	"time"

	"github.com/lbryio/lbry.go/errors"
	"github.com/lbryio/lbry.go/stopOnce"
	"github.com/lbryio/reflector.go/dht/bits"

	log "github.com/sirupsen/logrus"
)

// TODO: iterativeFindValue may be stopping early. if it gets a response with one peer, it should keep going because other nodes may know about more peers that have that blob
// TODO: or, it should try a tcp handshake with peers as it finds them, to make sure they are still online and have the blob

type contactFinder struct {
	findValue bool // true if we're using findValue
	target    bits.Bitmap
	node      *Node

	stop *stopOnce.Stopper

	findValueMutex  *sync.Mutex
	findValueResult []Contact

	activeContactsMutex *sync.Mutex
	activeContacts      []Contact

	shortlistMutex *sync.Mutex
	shortlist      []Contact
	shortlistAdded map[bits.Bitmap]bool

	outstandingRequestsMutex *sync.RWMutex
	outstandingRequests      uint
}

func FindContacts(node *Node, target bits.Bitmap, findValue bool, upstreamStop stopOnce.Chan) ([]Contact, bool, error) {
	cf := &contactFinder{
		node:                node,
		target:              target,
		findValue:           findValue,
		findValueMutex:      &sync.Mutex{},
		activeContactsMutex: &sync.Mutex{},
		shortlistMutex:      &sync.Mutex{},
		shortlistAdded:      make(map[bits.Bitmap]bool),
		stop:                stopOnce.New(),
		outstandingRequestsMutex: &sync.RWMutex{},
	}
	if upstreamStop != nil {
		cf.stop.Link(upstreamStop)
	}
	return cf.Find()
}

func (cf *contactFinder) Stop() {
	cf.stop.Stop()
	cf.stop.Wait()
}

func (cf *contactFinder) Find() ([]Contact, bool, error) {
	if cf.findValue {
		log.Debugf("[%s] starting an iterative Find for the value %s", cf.node.id.HexShort(), cf.target.HexShort())
	} else {
		log.Debugf("[%s] starting an iterative Find for contacts near %s", cf.node.id.HexShort(), cf.target.HexShort())
	}
	cf.appendNewToShortlist(cf.node.rt.GetClosest(cf.target, alpha))
	if len(cf.shortlist) == 0 {
		return nil, false, errors.Err("no contacts in routing table")
	}

	for i := 0; i < alpha; i++ {
		cf.stop.Add(1)
		go func(i int) {
			defer cf.stop.Done()
			cf.iterationWorker(i + 1)
		}(i)
	}

	cf.stop.Wait()

	// TODO: what to do if we have less than K active contacts, shortlist is empty, but we
	// TODO: have other contacts in our routing table whom we have not contacted. prolly contact them

	var contacts []Contact
	var found bool
	if cf.findValue && len(cf.findValueResult) > 0 {
		contacts = cf.findValueResult
		found = true
	} else {
		contacts = cf.activeContacts
		if len(contacts) > bucketSize {
			contacts = contacts[:bucketSize]
		}
	}

	cf.Stop()
	return contacts, found, nil
}

func (cf *contactFinder) iterationWorker(num int) {
	log.Debugf("[%s] starting worker %d", cf.node.id.HexShort(), num)
	defer func() { log.Debugf("[%s] stopping worker %d", cf.node.id.HexShort(), num) }()

	for {
		maybeContact := cf.popFromShortlist()
		if maybeContact == nil {
			// TODO: block if there are pending requests out from other workers. there may be more shortlist values coming
			log.Debugf("[%s] worker %d: no contacts in shortlist, waiting...", cf.node.id.HexShort(), num)
			time.Sleep(100 * time.Millisecond)
		} else {
			contact := *maybeContact

			if contact.ID.Equals(cf.node.id) {
				continue // cannot contact self
			}

			req := Request{Arg: &cf.target}
			if cf.findValue {
				req.Method = findValueMethod
			} else {
				req.Method = findNodeMethod
			}

			log.Debugf("[%s] worker %d: contacting %s", cf.node.id.HexShort(), num, contact.ID.HexShort())

			cf.incrementOutstanding()

			var res *Response
			resCh, cancel := cf.node.SendCancelable(contact, req)
			select {
			case res = <-resCh:
			case <-cf.stop.Ch():
				log.Debugf("[%s] worker %d: canceled", cf.node.id.HexShort(), num)
				cancel()
				return
			}

			if res == nil {
				// nothing to do, response timed out
				log.Debugf("[%s] worker %d: search canceled or timed out waiting for %s", cf.node.id.HexShort(), num, contact.ID.HexShort())
			} else if cf.findValue && res.FindValueKey != "" {
				log.Debugf("[%s] worker %d: got value", cf.node.id.HexShort(), num)
				cf.findValueMutex.Lock()
				cf.findValueResult = res.Contacts
				cf.findValueMutex.Unlock()
				cf.stop.Stop()
				return
			} else {
				log.Debugf("[%s] worker %d: got contacts", cf.node.id.HexShort(), num)
				cf.insertIntoActiveList(contact)
				cf.appendNewToShortlist(res.Contacts)
			}

			cf.decrementOutstanding() // this is all the way down here because we need to add to shortlist first
		}

		if cf.isSearchFinished() {
			log.Debugf("[%s] worker %d: search is finished", cf.node.id.HexShort(), num)
			cf.stop.Stop()
			return
		}
	}
}

func (cf *contactFinder) appendNewToShortlist(contacts []Contact) {
	cf.shortlistMutex.Lock()
	defer cf.shortlistMutex.Unlock()

	for _, c := range contacts {
		if _, ok := cf.shortlistAdded[c.ID]; !ok {
			cf.shortlist = append(cf.shortlist, c)
			cf.shortlistAdded[c.ID] = true
		}
	}

	sortInPlace(cf.shortlist, cf.target)
}

func (cf *contactFinder) popFromShortlist() *Contact {
	cf.shortlistMutex.Lock()
	defer cf.shortlistMutex.Unlock()

	if len(cf.shortlist) == 0 {
		return nil
	}

	first := cf.shortlist[0]
	cf.shortlist = cf.shortlist[1:]
	return &first
}

func (cf *contactFinder) insertIntoActiveList(contact Contact) {
	cf.activeContactsMutex.Lock()
	defer cf.activeContactsMutex.Unlock()

	inserted := false
	for i, n := range cf.activeContacts {
		// 5000ft: insert contact into sorted active contacts list
		// Detail: if diff between new contact id and the target id has fewer changes than the n contact from target
		//	it should be inserted in between the previous and current.
		if contact.ID.Xor(cf.target).Less(n.ID.Xor(cf.target)) {
			cf.activeContacts = append(cf.activeContacts[:i], append([]Contact{contact}, cf.activeContacts[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		cf.activeContacts = append(cf.activeContacts, contact)
	}
}

func (cf *contactFinder) isSearchFinished() bool {
	if cf.findValue && len(cf.findValueResult) > 0 {
		return true
	}

	select {
	case <-cf.stop.Ch():
		return true
	default:
	}

	if !cf.areRequestsOutstanding() {
		cf.shortlistMutex.Lock()
		defer cf.shortlistMutex.Unlock()

		if len(cf.shortlist) == 0 {
			return true
		}

		cf.activeContactsMutex.Lock()
		defer cf.activeContactsMutex.Unlock()

		if len(cf.activeContacts) >= bucketSize && cf.activeContacts[bucketSize-1].ID.Xor(cf.target).Less(cf.shortlist[0].ID.Xor(cf.target)) {
			// we have at least K active contacts, and we don't have any closer contacts to ping
			return true
		}
	}

	return false
}

func (cf *contactFinder) incrementOutstanding() {
	cf.outstandingRequestsMutex.Lock()
	defer cf.outstandingRequestsMutex.Unlock()
	cf.outstandingRequests++
}
func (cf *contactFinder) decrementOutstanding() {
	cf.outstandingRequestsMutex.Lock()
	defer cf.outstandingRequestsMutex.Unlock()
	if cf.outstandingRequests > 0 {
		cf.outstandingRequests--
	}
}
func (cf *contactFinder) areRequestsOutstanding() bool {
	cf.outstandingRequestsMutex.RLock()
	defer cf.outstandingRequestsMutex.RUnlock()
	return cf.outstandingRequests > 0
}

func sortInPlace(contacts []Contact, target bits.Bitmap) {
	toSort := make([]sortedContact, len(contacts))

	for i, n := range contacts {
		toSort[i] = sortedContact{n, n.ID.Xor(target)}
	}

	sort.Sort(byXorDistance(toSort))

	for i, c := range toSort {
		contacts[i] = c.contact
	}
}
