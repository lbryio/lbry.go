package dht

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/lbryio/errors.go"
	"github.com/lbryio/lbry.go/stopOnce"

	log "github.com/sirupsen/logrus"
)

// TODO: iterativeFindValue may be stopping early. if it gets a response with one peer, it should keep going because other nodes may know about more peers that have that blob
// TODO: or, it should try a tcp handshake with peers as it finds them, to make sure they are still online and have the blob

type contactFinder struct {
	findValue bool // true if we're using findValue
	target    Bitmap
	node      *Node

	done   *stopOnce.Stopper
	doneWG *sync.WaitGroup

	findValueMutex  *sync.Mutex
	findValueResult []Contact

	activeContactsMutex *sync.Mutex
	activeContacts      []Contact

	shortlistMutex *sync.Mutex
	shortlist      []Contact
	shortlistAdded map[Bitmap]bool

	outstandingRequestsMutex *sync.RWMutex
	outstandingRequests      uint
}

type findNodeResponse struct {
	Found    bool
	Contacts []Contact
}

func newContactFinder(node *Node, target Bitmap, findValue bool) *contactFinder {
	return &contactFinder{
		node:                node,
		target:              target,
		findValue:           findValue,
		findValueMutex:      &sync.Mutex{},
		activeContactsMutex: &sync.Mutex{},
		shortlistMutex:      &sync.Mutex{},
		shortlistAdded:      make(map[Bitmap]bool),
		done:                stopOnce.New(),
		doneWG:              &sync.WaitGroup{},
		outstandingRequestsMutex: &sync.RWMutex{},
	}
}

func (cf *contactFinder) Cancel() {
	cf.done.Stop()
	cf.doneWG.Wait()
}

func (cf *contactFinder) Find() (findNodeResponse, error) {
	if cf.findValue {
		log.Debugf("[%s] starting an iterative Find for the value %s", cf.node.id.HexShort(), cf.target.HexShort())
	} else {
		log.Debugf("[%s] starting an iterative Find for contacts near %s", cf.node.id.HexShort(), cf.target.HexShort())
	}
	cf.appendNewToShortlist(cf.node.rt.GetClosest(cf.target, alpha))
	if len(cf.shortlist) == 0 {
		return findNodeResponse{}, errors.Err("no contacts in routing table")
	}

	for i := 0; i < alpha; i++ {
		cf.doneWG.Add(1)
		go func(i int) {
			defer cf.doneWG.Done()
			cf.iterationWorker(i + 1)
		}(i)
	}

	cf.doneWG.Wait()

	// TODO: what to do if we have less than K active contacts, shortlist is empty, but we
	// TODO: have other contacts in our routing table whom we have not contacted. prolly contact them

	result := findNodeResponse{}
	if cf.findValue && len(cf.findValueResult) > 0 {
		result.Found = true
		result.Contacts = cf.findValueResult
	} else {
		result.Contacts = cf.activeContacts
		if len(result.Contacts) > bucketSize {
			result.Contacts = result.Contacts[:bucketSize]
		}
	}

	return result, nil
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

			if contact.id.Equals(cf.node.id) {
				continue // cannot contact self
			}

			req := Request{Arg: &cf.target}
			if cf.findValue {
				req.Method = findValueMethod
			} else {
				req.Method = findNodeMethod
			}

			log.Debugf("[%s] worker %d: contacting %s", cf.node.id.HexShort(), num, contact.id.HexShort())

			cf.incrementOutstanding()

			var res *Response
			ctx, cancel := context.WithCancel(context.Background())
			resCh := cf.node.SendAsync(ctx, contact, req)
			select {
			case res = <-resCh:
			case <-cf.done.Chan():
				log.Debugf("[%s] worker %d: canceled", cf.node.id.HexShort(), num)
				cancel()
				return
			}

			if res == nil {
				// nothing to do, response timed out
				log.Debugf("[%s] worker %d: search canceled or timed out waiting for %s", cf.node.id.HexShort(), num, contact.id.HexShort())
			} else if cf.findValue && res.FindValueKey != "" {
				log.Debugf("[%s] worker %d: got value", cf.node.id.HexShort(), num)
				cf.findValueMutex.Lock()
				cf.findValueResult = res.Contacts
				cf.findValueMutex.Unlock()
				cf.done.Stop()
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
			cf.done.Stop()
			return
		}
	}
}

func (cf *contactFinder) appendNewToShortlist(contacts []Contact) {
	cf.shortlistMutex.Lock()
	defer cf.shortlistMutex.Unlock()

	for _, c := range contacts {
		if _, ok := cf.shortlistAdded[c.id]; !ok {
			cf.shortlist = append(cf.shortlist, c)
			cf.shortlistAdded[c.id] = true
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
		if contact.id.Xor(cf.target).Less(n.id.Xor(cf.target)) {
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
	case <-cf.done.Chan():
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

		if len(cf.activeContacts) >= bucketSize && cf.activeContacts[bucketSize-1].id.Xor(cf.target).Less(cf.shortlist[0].id.Xor(cf.target)) {
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

func sortInPlace(contacts []Contact, target Bitmap) {
	toSort := make([]sortedContact, len(contacts))

	for i, n := range contacts {
		toSort[i] = sortedContact{n, n.id.Xor(target)}
	}

	sort.Sort(byXorDistance(toSort))

	for i, c := range toSort {
		contacts[i] = c.contact
	}
}
