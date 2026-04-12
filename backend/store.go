package main

import (
	"fmt"
	"sync"
)

type documentStore struct {
	mu   sync.RWMutex
	docs map[string]*Document
}

func newStore() *documentStore {
	return &documentStore{docs: make(map[string]*Document)}
}

func (s *documentStore) save(doc *Document) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[doc.ID] = doc
}

func (s *documentStore) get(id string) (*Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	doc, ok := s.docs[id]
	if !ok {
		return nil, fmt.Errorf("document %q not found", id)
	}
	return doc, nil
}
