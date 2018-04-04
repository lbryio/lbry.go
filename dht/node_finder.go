package dht

import (
	"context"
	"sync"

	"github.com/lbryio/errors.go"
	"github.com/lbryio/lbry.go/stopOnce"

	log "github.com/sirupsen/logrus"
)

type nodeFinder struct {
	findValue bool // true if we're using findValue
	target    bitmap
	dht       *DHT

	done *stopOnce.Stopper

	findValueMutex  *sync.Mutex
	findValueResult []Node

	activeNodesMutex *sync.Mutex
	activeNodes      []Node

	shortlistContactedMutex *sync.Mutex
	shortlist               []Node
	contacted               map[bitmap]bool
}

type findNodeResponse struct {
	Found bool
	Nodes []Node
}

func newNodeFinder(dht *DHT, target bitmap, findValue bool) *nodeFinder {
	return &nodeFinder{
		dht:                     dht,
		target:                  target,
		findValue:               findValue,
		findValueMutex:          &sync.Mutex{},
		activeNodesMutex:        &sync.Mutex{},
		shortlistContactedMutex: &sync.Mutex{},
		contacted:               make(map[bitmap]bool),
		done:                    stopOnce.New(),
	}
}

func (nf *nodeFinder) Find() (findNodeResponse, error) {
	log.Debugf("[%s] starting an iterative Find() for %s (findValue is %t)", nf.dht.node.id.HexShort(), nf.target.HexShort(), nf.findValue)
	nf.appendNewToShortlist(nf.dht.rt.GetClosest(nf.target, alpha))
	if len(nf.shortlist) == 0 {
		return findNodeResponse{}, errors.Err("no nodes in routing table")
	}

	wg := &sync.WaitGroup{}

	for i := 0; i < alpha; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			nf.iterationWorker(i + 1)
		}(i)
	}

	wg.Wait()

	// TODO: what to do if we have less than K active nodes, shortlist is empty, but we
	// TODO: have other nodes in our routing table whom we have not contacted. prolly contact them?

	result := findNodeResponse{}
	if nf.findValue && len(nf.findValueResult) > 0 {
		result.Found = true
		result.Nodes = nf.findValueResult
	} else {
		result.Nodes = nf.activeNodes
		if len(result.Nodes) > bucketSize {
			result.Nodes = result.Nodes[:bucketSize]
		}
	}

	return result, nil
}

func (nf *nodeFinder) iterationWorker(num int) {
	log.Debugf("[%s] starting worker %d", nf.dht.node.id.HexShort(), num)
	defer func() { log.Debugf("[%s] stopping worker %d", nf.dht.node.id.HexShort(), num) }()

	for {
		maybeNode := nf.popFromShortlist()
		if maybeNode == nil {
			// TODO: block if there are pending requests out from other workers. there may be more shortlist values coming
			log.Debugf("[%s] no more nodes in shortlist", nf.dht.node.id.HexShort())
			return
		}
		node := *maybeNode

		if node.id.Equals(nf.dht.node.id) {
			continue // cannot contact self
		}

		req := Request{Args: []string{nf.target.RawString()}}
		if nf.findValue {
			req.Method = findValueMethod
		} else {
			req.Method = findNodeMethod
		}

		log.Debugf("[%s] contacting %s", nf.dht.node.id.HexShort(), node.id.HexShort())

		var res *Response
		ctx, cancel := context.WithCancel(context.Background())
		resCh := nf.dht.tm.SendAsync(ctx, node, req)
		select {
		case res = <-resCh:
		case <-nf.done.Chan():
			log.Debugf("[%s] worker %d: canceled", nf.dht.node.id.HexShort(), num)
			cancel()
			return
		}

		if res == nil {
			// nothing to do, response timed out
		} else if nf.findValue && res.FindValueKey != "" {
			log.Debugf("[%s] worker %d: got value", nf.dht.node.id.HexShort(), num)
			nf.findValueMutex.Lock()
			nf.findValueResult = res.FindNodeData
			nf.findValueMutex.Unlock()
			nf.done.Stop()
			return
		} else {
			log.Debugf("[%s] worker %d: got more contacts", nf.dht.node.id.HexShort(), num)
			nf.insertIntoActiveList(node)
			nf.appendNewToShortlist(res.FindNodeData)
		}

		if nf.isSearchFinished() {
			log.Debugf("[%s] worker %d: search is finished", nf.dht.node.id.HexShort(), num)
			nf.done.Stop()
			return
		}
	}
}

func (nf *nodeFinder) appendNewToShortlist(nodes []Node) {
	nf.shortlistContactedMutex.Lock()
	defer nf.shortlistContactedMutex.Unlock()

	notContacted := []Node{}
	for _, n := range nodes {
		if _, ok := nf.contacted[n.id]; !ok {
			notContacted = append(notContacted, n)
		}
	}

	nf.shortlist = append(nf.shortlist, notContacted...)
	sortNodesInPlace(nf.shortlist, nf.target)
}

func (nf *nodeFinder) popFromShortlist() *Node {
	nf.shortlistContactedMutex.Lock()
	defer nf.shortlistContactedMutex.Unlock()

	if len(nf.shortlist) == 0 {
		return nil
	}

	first := nf.shortlist[0]
	nf.shortlist = nf.shortlist[1:]
	nf.contacted[first.id] = true
	return &first
}

func (nf *nodeFinder) insertIntoActiveList(node Node) {
	nf.activeNodesMutex.Lock()
	defer nf.activeNodesMutex.Unlock()

	inserted := false
	for i, n := range nf.activeNodes {
		if node.id.Xor(nf.target).Less(n.id.Xor(nf.target)) {
			nf.activeNodes = append(nf.activeNodes[:i], append([]Node{node}, nf.activeNodes[i:]...)...)
			inserted = true
		}
	}
	if !inserted {
		nf.activeNodes = append(nf.activeNodes, node)
	}
}

func (nf *nodeFinder) isSearchFinished() bool {
	if nf.findValue && len(nf.findValueResult) > 0 {
		return true
	}

	select {
	case <-nf.done.Chan():
		return true
	default:
	}

	nf.shortlistContactedMutex.Lock()
	defer nf.shortlistContactedMutex.Unlock()

	if len(nf.shortlist) == 0 {
		return true
	}

	nf.activeNodesMutex.Lock()
	defer nf.activeNodesMutex.Unlock()

	if len(nf.activeNodes) >= bucketSize && nf.activeNodes[bucketSize-1].id.Xor(nf.target).Less(nf.shortlist[0].id.Xor(nf.target)) {
		// we have at least K active nodes, and we don't have any closer nodes yet to contact
		return true
	}

	return false
}
