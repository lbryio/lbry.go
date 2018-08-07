package dht

import (
	"sync"
	"time"

	"github.com/lbryio/lbry.go/crypto"
	"github.com/lbryio/lbry.go/errors"
	"github.com/lbryio/lbry.go/stop"
	"github.com/lbryio/reflector.go/dht/bits"

	"github.com/sirupsen/logrus"
	"github.com/uber-go/atomic"
)

// TODO: iterativeFindValue may be stopping early. if it gets a response with one peer, it should keep going because other nodes may know about more peers that have that blob
// TODO: or, it should try a tcp handshake with peers as it finds them, to make sure they are still online and have the blob

var cfLog *logrus.Logger

func init() {
	cfLog = logrus.StandardLogger()
}

func NodeFinderUseLogger(l *logrus.Logger) {
	cfLog = l
}

type contactFinder struct {
	findValue bool // true if we're using findValue
	target    bits.Bitmap
	node      *Node

	grp *stop.Group

	findValueMutex  *sync.Mutex
	findValueResult []Contact

	activeContactsMutex *sync.Mutex
	activeContacts      []Contact

	shortlistMutex *sync.Mutex
	shortlist      []Contact
	shortlistAdded map[bits.Bitmap]bool

	closestContactMutex *sync.RWMutex
	closestContact      *Contact
	notGettingCloser    *atomic.Bool
}

func FindContacts(node *Node, target bits.Bitmap, findValue bool, parentGrp *stop.Group) ([]Contact, bool, error) {
	cf := &contactFinder{
		node:                node,
		target:              target,
		findValue:           findValue,
		findValueMutex:      &sync.Mutex{},
		activeContactsMutex: &sync.Mutex{},
		shortlistMutex:      &sync.Mutex{},
		shortlistAdded:      make(map[bits.Bitmap]bool),
		grp:                 stop.New(parentGrp),
		closestContactMutex: &sync.RWMutex{},
		notGettingCloser:    atomic.NewBool(false),
	}

	return cf.Find()
}

func (cf *contactFinder) Stop() {
	cf.grp.StopAndWait()
}

func (cf *contactFinder) Find() ([]Contact, bool, error) {
	if cf.findValue {
		cf.debug("starting iterativeFindValue")
	} else {
		cf.debug("starting iterativeFindNode")
	}

	cf.appendNewToShortlist(cf.node.rt.GetClosest(cf.target, alpha))
	if len(cf.shortlist) == 0 {
		return nil, false, errors.Err("[%s] find %s: no contacts in routing table", cf.node.id.HexShort(), cf.target.HexShort())
	}

	go cf.cycle(false)
	timeout := 5 * time.Second
CycleLoop:
	for {
		select {
		case <-time.After(timeout):
			go cf.cycle(false)
		case <-cf.grp.Ch():
			break CycleLoop
		}
	}

	// TODO: what to do if we have less than K active contacts, shortlist is empty, but we have other contacts in our routing table whom we have not contacted. prolly contact them

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

// cycle does a single cycle of sending alpha probes and checking results against closestNode
func (cf *contactFinder) cycle(bigCycle bool) {
	cycleID := crypto.RandString(6)
	if bigCycle {
		cf.debug("LAUNCHING CYCLE %s, AND ITS A BIG CYCLE", cycleID)
	} else {
		cf.debug("LAUNCHING CYCLE %s", cycleID)
	}
	defer cf.debug("CYCLE %s DONE", cycleID)

	cf.closestContactMutex.RLock()
	closestContact := cf.closestContact
	cf.closestContactMutex.RUnlock()

	var wg sync.WaitGroup
	ch := make(chan *Contact)

	limit := alpha
	if bigCycle {
		limit = bucketSize
	}

	for i := 0; i < limit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch <- cf.probe(cycleID)
		}()
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	foundCloser := false
	for {
		c, more := <-ch
		if !more {
			break
		}
		if c != nil && (closestContact == nil || cf.target.Closer(c.ID, closestContact.ID)) {
			if closestContact != nil {
				cf.debug("|%s| best contact improved: %s -> %s", cycleID, closestContact.ID.HexShort(), c.ID.HexShort())
			} else {
				cf.debug("|%s| best contact starting at %s", cycleID, c.ID.HexShort())
			}
			foundCloser = true
			closestContact = c
		}
	}

	if cf.isSearchFinished() {
		cf.grp.Stop()
		return
	}

	if foundCloser {
		cf.closestContactMutex.Lock()
		// have to check again after locking in case other probes found a closer one in the meantime
		if cf.closestContact == nil || cf.target.Closer(closestContact.ID, cf.closestContact.ID) {
			cf.closestContact = closestContact
		}
		cf.closestContactMutex.Unlock()
		go cf.cycle(false)
	} else if !bigCycle {
		cf.debug("|%s| no improvement, running big cycle", cycleID)
		go cf.cycle(true)
	} else {
		// big cycle ran and there was no improvement, so we're done
		cf.debug("|%s| big cycle ran, still no improvement", cycleID)
		cf.notGettingCloser.Store(true)
	}
}

// probe sends a single probe, updates the lists, and returns the closest contact it found
func (cf *contactFinder) probe(cycleID string) *Contact {
	maybeContact := cf.popFromShortlist()
	if maybeContact == nil {
		cf.debug("|%s| no contacts in shortlist, returning", cycleID)
		return nil
	}

	c := *maybeContact

	if c.ID.Equals(cf.node.id) {
		return nil
	}

	cf.debug("|%s| probe %s: launching", cycleID, c.ID.HexShort())

	req := Request{Arg: &cf.target}
	if cf.findValue {
		req.Method = findValueMethod
	} else {
		req.Method = findNodeMethod
	}

	var res *Response
	resCh := cf.node.SendAsync(c, req)
	select {
	case res = <-resCh:
	case <-cf.grp.Ch():
		cf.debug("|%s| probe %s: canceled", cycleID, c.ID.HexShort())
		return nil
	}

	if res == nil {
		cf.debug("|%s| probe %s: req canceled or timed out", cycleID, c.ID.HexShort())
		return nil
	}

	if cf.findValue && res.FindValueKey != "" {
		cf.debug("|%s| probe %s: got value", cycleID, c.ID.HexShort())
		cf.findValueMutex.Lock()
		cf.findValueResult = res.Contacts
		cf.findValueMutex.Unlock()
		cf.grp.Stop()
		return nil
	}

	cf.debug("|%s| probe %s: got %s", cycleID, c.ID.HexShort(), res.argsDebug())
	cf.insertIntoActiveList(c)
	cf.appendNewToShortlist(res.Contacts)

	cf.activeContactsMutex.Lock()
	contacts := cf.activeContacts
	if len(contacts) > bucketSize {
		contacts = contacts[:bucketSize]
	}
	contactsStr := ""
	for _, c := range contacts {
		contactsStr += c.ID.HexShort() + ", "
	}
	cf.activeContactsMutex.Unlock()

	return cf.closest(res.Contacts...)
}

// appendNewToShortlist appends any new contacts to the shortlist and sorts it by distance
// contacts that have already been added to the shortlist in the past are ignored
func (cf *contactFinder) appendNewToShortlist(contacts []Contact) {
	cf.shortlistMutex.Lock()
	defer cf.shortlistMutex.Unlock()

	for _, c := range contacts {
		if _, ok := cf.shortlistAdded[c.ID]; !ok {
			cf.shortlist = append(cf.shortlist, c)
			cf.shortlistAdded[c.ID] = true
		}
	}

	sortByDistance(cf.shortlist, cf.target)
}

// popFromShortlist pops the first contact off the shortlist and returns it
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

// insertIntoActiveList inserts the contact into appropriate place in the list of active contacts (sorted by distance)
func (cf *contactFinder) insertIntoActiveList(contact Contact) {
	cf.activeContactsMutex.Lock()
	defer cf.activeContactsMutex.Unlock()

	inserted := false
	for i, n := range cf.activeContacts {
		if cf.target.Closer(contact.ID, n.ID) {
			cf.activeContacts = append(cf.activeContacts[:i], append([]Contact{contact}, cf.activeContacts[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		cf.activeContacts = append(cf.activeContacts, contact)
	}
}

// isSearchFinished returns true if the search is done and should be stopped
func (cf *contactFinder) isSearchFinished() bool {
	if cf.findValue && len(cf.findValueResult) > 0 {
		return true
	}

	select {
	case <-cf.grp.Ch():
		return true
	default:
	}

	if cf.notGettingCloser.Load() {
		return true
	}

	cf.activeContactsMutex.Lock()
	defer cf.activeContactsMutex.Unlock()
	return len(cf.activeContacts) >= bucketSize
}

func (cf *contactFinder) debug(format string, args ...interface{}) {
	args = append([]interface{}{cf.node.id.HexShort()}, append([]interface{}{cf.target.HexShort()}, args...)...)
	cfLog.Debugf("[%s] find %s: "+format, args...)
}

func (cf *contactFinder) closest(contacts ...Contact) *Contact {
	if len(contacts) == 0 {
		return nil
	}
	closest := contacts[0]
	for _, c := range contacts {
		if cf.target.Closer(c.ID, closest.ID) {
			closest = c
		}
	}
	return &closest
}
