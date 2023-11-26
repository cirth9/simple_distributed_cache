package simpleCache

import (
	"simpleCache/lru"
	"sync"
)

type cache struct {
	mu         sync.Mutex
	cache      *lru.Cache
	cacheBytes int64
}

func (c *cache) get(key string) (byteView *ByteView, ok1 bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cache == nil {
		return &ByteView{}, false
	}
	if val, ok := c.cache.Get(key); ok {
		return val.(*ByteView), ok
	}
	return
}

func (c *cache) add(key string, val *ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cache == nil {
		c.cache = lru.NewCache(c.cacheBytes, nil)
	}
	c.cache.Add(key, val)
}
