package consistentHash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

const DefaultReplicas = 3

type Hash func([]byte) uint32
type ConsistentHash struct {
	hash     Hash
	replicas int
	keys     []int
	hashMap  map[int]string
}

func NewConsistentHash(replicas int, fn Hash) *ConsistentHash {
	m := &ConsistentHash{
		replicas: replicas,
		hash:     fn,
		keys:     make([]int, 0),
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

func (h *ConsistentHash) Add(keys ...string) {
	for _, v := range keys {
		for i := 0; i < h.replicas; i++ {
			hashNumber := h.hash([]byte(strconv.Itoa(i) + v))
			h.keys = append(h.keys, int(hashNumber))
			h.hashMap[int(hashNumber)] = v
		}
	}
	sort.Ints(h.keys)
}

func (h *ConsistentHash) Get(key string) string {
	if len(h.keys) == 0 {
		return ""
	}
	hashNumber := int(h.hash([]byte(key)))
	n := sort.Search(len(h.keys), func(i int) bool {
		return h.keys[i] >= hashNumber
	})
	return h.hashMap[h.keys[n%len(h.keys)]]
}
