package dht

import (
	"sync"
	"time"
)

type peer struct {
	node                *Node
	lastPublished       time.Time
	originallyPublished time.Time
	originalPublisherID bitmap
}

type peerStore struct {
	data map[bitmap][]peer
	lock sync.RWMutex
}

func newPeerStore() *peerStore {
	return &peerStore{
		data: map[bitmap][]peer{},
	}
}

func (s *peerStore) Insert(key bitmap, node *Node, lastPublished, originallyPublished time.Time, originaPublisherID bitmap) {
	s.lock.Lock()
	defer s.lock.Unlock()
	newPeer := peer{node: node, lastPublished: lastPublished, originallyPublished: originallyPublished, originalPublisherID: originaPublisherID}
	_, ok := s.data[key]
	if !ok {
		s.data[key] = []peer{newPeer}
	} else {
		s.data[key] = append(s.data[key], newPeer)
	}
}

func (s *peerStore) GetNodes(key bitmap) []*Node {
	s.lock.RLock()
	defer s.lock.RUnlock()
	nodes := []*Node{}
	if peers, ok := s.data[key]; ok {
		for _, p := range peers {
			nodes = append(nodes, p.node)
		}
	}
	return nodes
}
