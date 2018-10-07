package lruish

import (
	"errors"
	"sync"
)

type Cache interface {
	Add(key, value interface{}) bool
	Get(key interface{}) (value interface{}, ok bool)
	Contains(key interface{}) bool
	Peek(key interface{}) (value interface{}, ok bool)
	ContainsOrAdd(key, value interface{}) (ok, evicted bool)
	Remove(key interface{}) bool
	Keys() []interface{}
	Len() int
}

// SynchedLRU is a thread-safe fixed size LRU cache.
type SynchedLRU struct {
	lru  Cache
	lock sync.RWMutex
}

// NewSynched creates an multi-thread safe LRU cache of the given size.
func NewSynched(size int) (Cache, error) {
	lru, err := NewUnsyched(size)
	if err != nil {
		return nil, err
	}
	c := &SynchedLRU{
		lru: lru,
	}
	return c, nil
}

// Add adds a value to the cache.  Returns true if an eviction occurred.
func (c *SynchedLRU) Add(key, value interface{}) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.lru.Add(key, value)
}

// Get looks up a key's value from the cache.
func (c *SynchedLRU) Get(key interface{}) (value interface{}, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.lru.Get(key)
}

// Contains checks if a key is in the cache, without updating the
// recent-ness or deleting it for being stale.
func (c *SynchedLRU) Contains(key interface{}) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.lru.Contains(key)
}

// Peek returns the key value (or undefined if not found) without updating
// the "recently used"-ness of the key.
func (c *SynchedLRU) Peek(key interface{}) (value interface{}, ok bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.lru.Peek(key)
}
func (c *SynchedLRU) Keys() []interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.lru.Keys()
}

// Len returns the number of items in the cache.
func (c *SynchedLRU) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.lru.Len()
}

// ContainsOrAdd checks if a key is in the cache  without updating the
// recent-ness or deleting it for being stale,  and if not, adds the value.
// Returns whether found and whether an eviction occurred.
func (c *SynchedLRU) ContainsOrAdd(key, value interface{}) (ok, evicted bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.ContainsOrAdd(key, value)
}

// Remove removes the provided key from the cache.
func (c *SynchedLRU) Remove(key interface{}) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.lru.Remove(key)

}

// NewUnsynched creates an non-multi-thread safe LRU cache of the given size.
func NewUnsynched(size int) (Cache, error) {
	if size <= 0 {
		return nil, errors.New("must provide a positive size")
	}

	c := &lruish{
		size:  size,
		head:  0,
		items: make(map[interface{}]*lruElem),
		ring:  make([]*lruElem, size),
	}
	return c, nil
}

type lruElem struct {
	// The value stored with this element.
	value interface{}
	key   interface{}
	index int
}

type lruish struct {
	size  int
	items map[interface{}]*lruElem
	head  int
	ring  []*lruElem
}

// ContainsOrAdd checks if a key is in the cache  without updating the
// recent-ness or deleting it for being stale,  and if not, adds the value.
// Returns whether found and whether an eviction occurred.
func (c *lruish) ContainsOrAdd(key, value interface{}) (ok, evicted bool) {
	if c.Contains(key) {
		return true, false
	}
	evicted = c.Add(key, value)
	return false, evicted
}

// Keys returns the keys, unordered
func (c *lruish) Keys() []interface{} {
	keys := make([]interface{}, len(c.items))
	i := 0
	for k := range c.items {
		keys[i] = k
		i++
	}
	return keys
}

func (c *lruish) Len() int {
	return len(c.items)
}

func (c *lruish) promote(ent *lruElem) {
	curIndex := ent.index
	// Calculate the new position for this item
	position := curIndex - c.head
	if position < 0 {
		position += c.size
	}
	// Calculate new index to place this item at
	newIndex := (c.head + position/2) % c.size
	// Update the downgraded item, if non-nil (could be a hole in the ring)
	if c.ring[newIndex] != nil {
		c.ring[newIndex].index = curIndex
	}
	// Update the promoted item
	ent.index = newIndex
	// Swap them
	c.ring[curIndex], c.ring[newIndex] = c.ring[newIndex], c.ring[curIndex]
}

func (c *lruish) Get(key interface{}) (interface{}, bool) {
	if ent, ok := c.items[key]; ok {
		c.promote(ent)
		return ent.value, true
	}
	return nil, false
}

// Add adds a value to the cache.  Returns true if an eviction occurred.
func (c *lruish) Add(key, value interface{}) bool {
	// Check for existing item
	if ent, ok := c.items[key]; ok {
		c.promote(ent)
		ent.value = value
		return false
	}
	// Add a new item
	// new head position is h-1
	c.head--
	if c.head < 0 {
		c.head += c.size
	}
	// new tail position is h -2
	tailIndex := c.head - 1
	if tailIndex < 0 {
		tailIndex += c.size
	}
	if toDelete := c.ring[tailIndex]; toDelete != nil {
		delete(c.items, toDelete.key)
	}
	ent := &lruElem{value: value, key: key, index: c.head}
	c.items[key] = ent
	c.ring[c.head] = ent
	return true
}

// Check if a key is in the cache, without updating the recent-ness
// or deleting it for being stale.
func (c *lruish) Contains(key interface{}) (ok bool) {
	_, ok = c.items[key]
	return ok
}

// Returns the key value (or undefined if not found) without updating
// the "recently used"-ness of the key.
func (c *lruish) Peek(key interface{}) (interface{}, bool) {
	if ent, ok := c.items[key]; ok {
		return ent.value, true
	}
	return nil, false
}

// Purge is used to completely clear the cache
func (c *lruish) Purge() {
	c.items = make(map[interface{}]*lruElem)
	c.ring = make([]*lruElem, c.size)
	c.head = 0
}

// Remove removes the provided key from the cache, returning if the
// key was contained.
func (c *lruish) Remove(key interface{}) bool {
	if ent, ok := c.items[key]; ok {
		delete(c.items, key)
		// We'll leave a whole in the ring, but
		// it will gradually be moved out
		c.ring[ent.index] = nil
		return true
	}
	return false
}
