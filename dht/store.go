package dht

import "sync"

type peer struct {
	node Node
	//<lastPublished>,
	//<originallyPublished>
	//	<originalPublisherID>
}

type peerStore struct {
	// map of blob hashes to (map of node IDs to bools)
	nodeIDs map[bitmap]map[bitmap]bool
	// map of node IDs to peers
	nodeInfo map[bitmap]peer
	lock     sync.RWMutex
}

func newPeerStore() *peerStore {
	return &peerStore{
		nodeIDs:  make(map[bitmap]map[bitmap]bool),
		nodeInfo: make(map[bitmap]peer),
	}
}

func (s *peerStore) Upsert(blobHash bitmap, node Node) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.nodeIDs[blobHash]; !ok {
		s.nodeIDs[blobHash] = make(map[bitmap]bool)
	}
	s.nodeIDs[blobHash][node.id] = true
	s.nodeInfo[node.id] = peer{node: node}
}

func (s *peerStore) Get(blobHash bitmap) []Node {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var nodes []Node
	if ids, ok := s.nodeIDs[blobHash]; ok {
		for id := range ids {
			peer, ok := s.nodeInfo[id]
			if !ok {
				panic("node id in IDs list, but not in nodeInfo")
			}
			nodes = append(nodes, peer.node)
		}
	}
	return nodes
}

func (s *peerStore) CountKnownNodes() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.nodeInfo)
}
