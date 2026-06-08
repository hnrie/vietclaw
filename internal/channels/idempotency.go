package channels

import (
	"sync"
	"time"
)

type TTLGuard struct {
	ttl   time.Duration
	mu    sync.Mutex
	items map[string]time.Time
}

func NewTTLGuard(ttl time.Duration) *TTLGuard {
	return &TTLGuard{ttl: ttl, items: map[string]time.Time{}}
}

func (g *TTLGuard) Seen(key string) bool {
	now := time.Now()
	g.mu.Lock()
	defer g.mu.Unlock()
	for item, expires := range g.items {
		if now.After(expires) {
			delete(g.items, item)
		}
	}
	if expires, ok := g.items[key]; ok && now.Before(expires) {
		return false
	}
	g.items[key] = now.Add(g.ttl)
	return true
}
