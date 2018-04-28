package dht

import "sync"

type peer struct {
	contact Contact
	//<lastPublished>,
	//<originallyPublished>
	//	<originalPublisherID>
}

type peerStore struct {
	// map of blob hashes to (map of node IDs to bools)
	hashes map[Bitmap]map[Bitmap]bool
	// stores the peers themselves, so they can be updated in one place
	peers map[Bitmap]peer
	lock  sync.RWMutex
}

func newPeerStore() *peerStore {
	return &peerStore{
		hashes: make(map[Bitmap]map[Bitmap]bool),
		peers:  make(map[Bitmap]peer),
	}
}

func (s *peerStore) Upsert(blobHash Bitmap, contact Contact) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.hashes[blobHash]; !ok {
		s.hashes[blobHash] = make(map[Bitmap]bool)
	}
	s.hashes[blobHash][contact.id] = true
	s.peers[contact.id] = peer{contact: contact}
}

func (s *peerStore) Get(blobHash Bitmap) []Contact {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var contacts []Contact
	if ids, ok := s.hashes[blobHash]; ok {
		for id := range ids {
			peer, ok := s.peers[id]
			if !ok {
				panic("node id in IDs list, but not in nodeInfo")
			}
			contacts = append(contacts, peer.contact)
		}
	}
	return contacts
}

func (s *peerStore) RemoveTODO(contact Contact) {
	// TODO: remove peer from everywhere
}

func (s *peerStore) CountStoredHashes() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.hashes)
}
