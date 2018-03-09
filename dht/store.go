package dht

import "sync"

type peer struct {
	node Node
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

func (s *peerStore) Insert(key string, node Node) {
	s.lock.Lock()
	defer s.lock.Unlock()
	newPeer := peer{node: node}
	_, ok := s.data[key]
	if !ok {
		s.data[key] = []peer{newPeer}
	} else {
		s.data[key] = append(s.data[key], newPeer)
	}
}

func (s *peerStore) Get(key string) []Node {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var nodes []Node
	if peers, ok := s.data[key]; ok {
		for _, p := range peers {
			nodes = append(nodes, p.node)
		}
	}
	return nodes
}
