package chat

import (
	"errors"
	"strings"
	"sync"
)

// The error returned when an added id already exists in the set.
var ErrIdTaken = errors.New("id already taken")

// The error returned when a requested item does not exist in the set.
var ErrItemMissing = errors.New("item does not exist")

// Id is a unique identifier for an item.
type Id string

// Item is an interface for items to store-able in the set
type Item interface {
	Id() Id
}

// Set with string lookup.
// TODO: Add trie for efficient prefix lookup?
type Set struct {
	lookup map[Id]Item
	sync.RWMutex
}

// NewSet creates a new set.
func NewSet() *Set {
	return &Set{
		lookup: map[Id]Item{},
	}
}

// Clear removes all items and returns the number removed.
func (s *Set) Clear() int {
	s.Lock()
	n := len(s.lookup)
	s.lookup = map[Id]Item{}
	s.Unlock()
	return n
}

// Len returns the size of the set right now.
func (s *Set) Len() int {
	return len(s.lookup)
}

// In checks if an item exists in this set.
func (s *Set) In(item Item) bool {
	s.RLock()
	_, ok := s.lookup[item.Id()]
	s.RUnlock()
	return ok
}

// Get returns an item with the given Id.
func (s *Set) Get(id Id) (Item, error) {
	s.RLock()
	item, ok := s.lookup[id]
	s.RUnlock()

	if !ok {
		return nil, ErrItemMissing
	}

	return item, nil
}

// Add item to this set if it does not exist already.
func (s *Set) Add(item Item) error {
	s.Lock()
	defer s.Unlock()

	_, found := s.lookup[item.Id()]
	if found {
		return ErrIdTaken
	}

	s.lookup[item.Id()] = item
	return nil
}

// Remove item from this set.
func (s *Set) Remove(item Item) error {
	s.Lock()
	defer s.Unlock()
	id := item.Id()
	_, found := s.lookup[id]
	if !found {
		return ErrItemMissing
	}
	delete(s.lookup, id)
	return nil
}

// Each loops over every item while holding a read lock and applies fn to each
// element.
func (s *Set) Each(fn func(item Item)) {
	s.RLock()
	for _, item := range s.lookup {
		fn(item)
	}
	s.RUnlock()
}

// ListPrefix returns a list of items with a prefix, case insensitive.
func (s *Set) ListPrefix(prefix string) []Item {
	r := []Item{}
	prefix = strings.ToLower(prefix)

	s.RLock()
	defer s.RUnlock()

	for id, item := range s.lookup {
		if !strings.HasPrefix(string(id), prefix) {
			continue
		}
		r = append(r, item)
	}

	return r
}