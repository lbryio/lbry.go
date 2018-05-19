package dht

import "sync"

type contactStore struct {
	// map of blob hashes to (map of node IDs to bools)
	hashes map[Bitmap]map[Bitmap]bool
	// stores the peers themselves, so they can be updated in one place
	contacts map[Bitmap]Contact
	lock     sync.RWMutex
}

func newStore() *contactStore {
	return &contactStore{
		hashes:   make(map[Bitmap]map[Bitmap]bool),
		contacts: make(map[Bitmap]Contact),
	}
}

func (s *contactStore) Upsert(blobHash Bitmap, contact Contact) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.hashes[blobHash]; !ok {
		s.hashes[blobHash] = make(map[Bitmap]bool)
	}
	s.hashes[blobHash][contact.ID] = true
	s.contacts[contact.ID] = contact
}

func (s *contactStore) Get(blobHash Bitmap) []Contact {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var contacts []Contact
	if ids, ok := s.hashes[blobHash]; ok {
		for id := range ids {
			contact, ok := s.contacts[id]
			if !ok {
				panic("node id in IDs list, but not in nodeInfo")
			}
			contacts = append(contacts, contact)
		}
	}
	return contacts
}

func (s *contactStore) RemoveTODO(contact Contact) {
	// TODO: remove peer from everywhere
}

func (s *contactStore) CountStoredHashes() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.hashes)
}
