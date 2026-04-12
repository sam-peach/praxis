package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type mappingStore struct {
	mu       sync.RWMutex
	data     map[string]*Mapping // keyed by normKey(customerPartNumber)
	filePath string
}

func newMappingStore(filePath string) (*mappingStore, error) {
	ms := &mappingStore{
		data:     make(map[string]*Mapping),
		filePath: filePath,
	}
	if err := ms.load(); err != nil {
		return nil, err
	}
	return ms, nil
}

// load reads the JSON file on startup. A missing file is not an error.
func (s *mappingStore) load() error {
	b, err := os.ReadFile(s.filePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read mappings file: %w", err)
	}
	var list []*Mapping
	if err := json.Unmarshal(b, &list); err != nil {
		return fmt.Errorf("parse mappings file: %w", err)
	}
	for _, m := range list {
		if k := normKey(m.CustomerPartNumber); k != "" {
			s.data[k] = m
		}
	}
	return nil
}

// persist writes the current state to the JSON file atomically.
// No-ops when filePath is empty (in-memory / test mode).
// Must be called with the write lock held (or from a method that holds it).
func (s *mappingStore) persist() error {
	if s.filePath == "" {
		return nil
	}
	list := make([]*Mapping, 0, len(s.data))
	for _, m := range s.data {
		list = append(list, m)
	}
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.filePath)
}

func normKey(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// save creates or updates a mapping and persists to disk.
func (s *mappingStore) save(m *Mapping) error {
	key := normKey(m.CustomerPartNumber)
	if key == "" {
		return fmt.Errorf("customerPartNumber is required")
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.data[key]; ok {
		// Preserve creation timestamp; update everything else.
		m.ID = existing.ID
		m.CreatedAt = existing.CreatedAt
	} else {
		if m.ID == "" {
			m.ID = newID()
		}
		m.CreatedAt = now
	}
	m.UpdatedAt = now
	if m.LastUsedAt.IsZero() {
		m.LastUsedAt = now
	}
	s.data[key] = m
	return s.persist()
}

// touchLastUsed updates LastUsedAt without changing other fields. Best-effort (no error returned).
func (s *mappingStore) touchLastUsed(customerPartNumber string) {
	key := normKey(customerPartNumber)
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.data[key]; ok {
		m.LastUsedAt = time.Now().UTC()
		_ = s.persist()
	}
}

func (s *mappingStore) lookup(customerPartNumber string) (*Mapping, bool) {
	key := normKey(customerPartNumber)
	if key == "" {
		return nil, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.data[key]
	return m, ok
}

func (s *mappingStore) all() []*Mapping {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Mapping, 0, len(s.data))
	for _, m := range s.data {
		out = append(out, m)
	}
	return out
}
