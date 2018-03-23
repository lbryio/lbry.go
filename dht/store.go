package dht

import "sync"

type peer struct {
	node Node
	//<lastPublished>,
	//<originallyPublished>
	//	<originalPublisherID>
}

type peerStore struct {
	nodeIDs  map[string]map[bitmap]bool
	nodeInfo map[bitmap]peer
	lock     sync.RWMutex
}

func newPeerStore() *peerStore {
	return &peerStore{
		nodeIDs:  make(map[string]map[bitmap]bool),
		nodeInfo: make(map[bitmap]peer),
	}
}

func (s *peerStore) Upsert(key string, node Node) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.nodeIDs[key]; !ok {
		s.nodeIDs[key] = make(map[bitmap]bool)
	}
	s.nodeIDs[key][node.id] = true
	s.nodeInfo[node.id] = peer{node: node}
}

func (s *peerStore) Get(key string) []Node {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var nodes []Node
	if ids, ok := s.nodeIDs[key]; ok {
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
