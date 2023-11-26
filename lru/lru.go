package lru

import "container/list"

type Cache struct {
	maxBytes int64
	nowBytes int64

	ll    *list.List
	cache map[string]*list.Element

	//某条记录被删除时的回调函数
	OnEvicted func(key string, value Value)
}

type entry struct {
	key string
	val Value
}

type Value interface {
	Len() int
}

// Len the number of cache entries
func (c *Cache) Len() int {
	return c.ll.Len()
}

func NewCache(maxBytes int64, onEvicted func(key string, value Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		nowBytes:  0,
		ll:        new(list.List),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

func (c *Cache) Add(key string, val Value) {
	if element, ok := c.cache[key]; ok {
		c.ll.MoveToFront(element)
		kv := element.Value.(*entry)
		c.nowBytes += int64(val.Len()) - int64(kv.val.Len())
		kv.val = val
	} else {
		front := c.ll.PushFront(&entry{key, val})
		c.cache[key] = front
		c.nowBytes += int64(len(key)) + int64(val.Len())
	}
	for c.maxBytes != 0 && c.nowBytes > c.maxBytes {
		c.RemoveOldest()
	}
}

func (c *Cache) Get(key string) (val Value, ok bool) {
	if element, ok1 := c.cache[key]; ok1 {
		c.ll.MoveToFront(element)
		kv := element.Value.(*entry)
		return kv.val, ok1
	}
	return
}

func (c *Cache) RemoveOldest() {
	oldest := c.ll.Back()
	if oldest != nil {
		c.ll.Remove(oldest)
		kv := oldest.Value.(*entry)
		delete(c.cache, kv.key)
		c.nowBytes -= int64(len(kv.key)) + int64(kv.val.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.val)
		}
	}
}
