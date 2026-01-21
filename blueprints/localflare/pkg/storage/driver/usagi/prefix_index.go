package usagi

import (
	"sort"
	"strings"
	"sync"
)

type prefixIndex struct {
	mu       sync.RWMutex
	maxDepth int
	items    map[string]*prefixList
}

type prefixList struct {
	keys []string
}

func newPrefixIndex(maxDepth int) *prefixIndex {
	if maxDepth < 1 {
		maxDepth = 1
	}
	return &prefixIndex{
		maxDepth: maxDepth,
		items:    make(map[string]*prefixList),
	}
}

func (p *prefixIndex) Add(key string) {
	for _, pref := range prefixesForKey(key, p.maxDepth) {
		p.insert(pref, key)
	}
}

func (p *prefixIndex) Remove(key string) {
	for _, pref := range prefixesForKey(key, p.maxDepth) {
		p.remove(pref, key)
	}
}

func (p *prefixIndex) Get(prefix string) ([]string, bool) {
	p.mu.RLock()
	list, ok := p.items[prefix]
	if !ok {
		p.mu.RUnlock()
		return nil, false
	}
	keys := append([]string(nil), list.keys...)
	p.mu.RUnlock()
	return keys, true
}

func (p *prefixIndex) Candidates(prefix string) ([]string, bool) {
	if prefix == "" {
		return nil, false
	}
	if keys, ok := p.Get(prefix); ok {
		return keys, true
	}
	parts := strings.Split(prefix, "/")
	if len(parts) <= 1 {
		return nil, false
	}
	if len(parts) > p.maxDepth {
		parts = parts[:p.maxDepth]
	}
	for depth := len(parts); depth >= 1; depth-- {
		candidate := strings.Join(parts[:depth], "/")
		if keys, ok := p.Get(candidate); ok {
			return keys, true
		}
	}
	return nil, false
}

func (p *prefixIndex) BuildFromIndex(entries map[string]*entry) {
	p.mu.Lock()
	p.items = make(map[string]*prefixList)
	for k := range entries {
		for _, pref := range prefixesForKey(k, p.maxDepth) {
			pl := p.items[pref]
			if pl == nil {
				pl = &prefixList{}
				p.items[pref] = pl
			}
			pl.keys = append(pl.keys, k)
		}
	}
	for _, pl := range p.items {
		sort.Strings(pl.keys)
		pl.keys = compactSorted(pl.keys)
	}
	p.mu.Unlock()
}

func (p *prefixIndex) insert(prefix, key string) {
	p.mu.Lock()
	pl := p.items[prefix]
	if pl == nil {
		pl = &prefixList{}
		p.items[prefix] = pl
	}
	idx := sort.SearchStrings(pl.keys, key)
	if idx < len(pl.keys) && pl.keys[idx] == key {
		p.mu.Unlock()
		return
	}
	pl.keys = append(pl.keys, "")
	copy(pl.keys[idx+1:], pl.keys[idx:])
	pl.keys[idx] = key
	p.mu.Unlock()
}

func (p *prefixIndex) remove(prefix, key string) {
	p.mu.Lock()
	pl := p.items[prefix]
	if pl == nil {
		p.mu.Unlock()
		return
	}
	idx := sort.SearchStrings(pl.keys, key)
	if idx < len(pl.keys) && pl.keys[idx] == key {
		pl.keys = append(pl.keys[:idx], pl.keys[idx+1:]...)
	}
	p.mu.Unlock()
}

func prefixesForKey(key string, maxDepth int) []string {
	if key == "" {
		return nil
	}
	parts := strings.Split(key, "/")
	if len(parts) == 0 {
		return nil
	}
	if len(parts) > maxDepth {
		parts = parts[:maxDepth]
	}
	prefixes := make([]string, 0, len(parts))
	var sb strings.Builder
	for i, part := range parts {
		if i > 0 {
			sb.WriteByte('/')
		}
		sb.WriteString(part)
		prefixes = append(prefixes, sb.String())
	}
	return prefixes
}

func compactSorted(keys []string) []string {
	if len(keys) == 0 {
		return keys
	}
	out := keys[:1]
	for i := 1; i < len(keys); i++ {
		if keys[i] != out[len(out)-1] {
			out = append(out, keys[i])
		}
	}
	return out
}
