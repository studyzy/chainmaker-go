package libp2ppubsub

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"sync"
)

// Whitelist is an interface for peer whitelisting.
type Whitelist interface {
	Add(peer.ID) bool
	Contains(peer.ID) bool
	Remove(peer.ID) bool
	Size() int
}

// MapWhitelist is a whitelist implementation using a perfect map
type MapWhitelist struct {
	m    map[peer.ID]struct{}
	lock sync.RWMutex
}

// NewMapBlacklist creates a new MapBlacklist
func NewMapWhitelist() Whitelist {
	return &MapWhitelist{m: make(map[peer.ID]struct{})}
}

func (b *MapWhitelist) Remove(p peer.ID) bool {
	b.lock.Lock()
	defer b.lock.Unlock()
	if _, ok := b.m[p]; ok {
		delete(b.m, p)
	}
	return true
}

func (b *MapWhitelist) Add(p peer.ID) bool {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.m[p] = struct{}{}
	return true
}

func (b *MapWhitelist) Contains(p peer.ID) bool {
	b.lock.RLock()
	defer b.lock.RUnlock()
	_, ok := b.m[p]
	return ok
}

func (b *MapWhitelist) Size() int {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return len(b.m)
}
