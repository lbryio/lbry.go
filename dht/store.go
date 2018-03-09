package dht

import "sync"

type peer struct {
	nodeID bitmap
}

type peerStore struct {
	data map[string][]peer
	lock sync.RWMutex
}

func newPeerStore() *peerStore {
	return &peerStore{
		data: make(map[string][]peer),
	}
}

func (s *peerStore) Insert(key string, nodeId bitmap) {
	s.lock.Lock()
	defer s.lock.Unlock()
	newPeer := peer{nodeID: nodeId}
	_, ok := s.data[key]
	if !ok {
		s.data[key] = []peer{newPeer}
	} else {
		s.data[key] = append(s.data[key], newPeer)
	}
}

func (s *peerStore) Get(key string) []bitmap {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var nodes []bitmap
	if peers, ok := s.data[key]; ok {
		for _, p := range peers {
			nodes = append(nodes, p.nodeID)
		}
	}
	return nodes
}
