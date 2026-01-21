package usagi

import (
	"container/list"
	"sync"
)

type smallCache struct {
	mu       sync.Mutex
	maxBytes int64
	maxItem  int64
	curBytes int64
	items    map[string]*list.Element
	order    *list.List
}

type cacheEntry struct {
	key  string
	data []byte
	size int64
}

func newSmallCache(maxBytes, maxItem int64) *smallCache {
	if maxBytes <= 0 || maxItem <= 0 {
		return nil
	}
	return &smallCache{
		maxBytes: maxBytes,
		maxItem:  maxItem,
		items:    make(map[string]*list.Element),
		order:    list.New(),
	}
}

func (c *smallCache) Get(key string) ([]byte, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if elem, ok := c.items[key]; ok {
		entry := elem.Value.(*cacheEntry)
		c.order.MoveToFront(elem)
		return append([]byte(nil), entry.data...), true
	}
	return nil, false
}

func (c *smallCache) Put(key string, data []byte) {
	if c == nil {
		return
	}
	sz := int64(len(data))
	if sz <= 0 || sz > c.maxItem {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if elem, ok := c.items[key]; ok {
		entry := elem.Value.(*cacheEntry)
		c.curBytes -= entry.size
		entry.data = append(entry.data[:0], data...)
		entry.size = sz
		c.curBytes += sz
		c.order.MoveToFront(elem)
		c.evict()
		return
	}
	entry := &cacheEntry{key: key, data: append([]byte(nil), data...), size: sz}
	elem := c.order.PushFront(entry)
	c.items[key] = elem
	c.curBytes += sz
	c.evict()
}

func (c *smallCache) Delete(key string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if elem, ok := c.items[key]; ok {
		entry := elem.Value.(*cacheEntry)
		c.curBytes -= entry.size
		delete(c.items, key)
		c.order.Remove(elem)
	}
}

func (c *smallCache) evict() {
	for c.curBytes > c.maxBytes {
		elem := c.order.Back()
		if elem == nil {
			return
		}
		entry := elem.Value.(*cacheEntry)
		delete(c.items, entry.key)
		c.curBytes -= entry.size
		c.order.Remove(elem)
	}
}
