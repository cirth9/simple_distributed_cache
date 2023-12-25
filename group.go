package simpleCache

import (
	"errors"
	"log"
	pb "simpleCache/cacheProtobuf"
	"simpleCache/singleFlight"
	"sync"
)

type Group struct {
	name      string
	getter    Getter
	mainCache cache
	peers     PeerPicker
	single    *singleFlight.Group
}

type Getter interface {
	Get(key string) ([]byte, error)
}

type GetterHandler func(key string) ([]byte, error)

func (f GetterHandler) Get(key string) ([]byte, error) {
	return f(key)
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

func NewGroup(groupName string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil getter")
	}
	mu.Lock()
	defer mu.Unlock()
	group := &Group{
		name:      groupName,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		single:    &singleFlight.Group{},
	}
	groups[groupName] = group
	groups[groupName].RegisterPeers(Pool)
	return group
}

func (g *Group) RegisterPeers(peer PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peer
}

func GetGroup(groupName string) *Group {
	mu.RLock()
	defer mu.RUnlock()
	if group, ok := groups[groupName]; ok {
		return group
	}
	return nil
}

func (g *Group) Get(key string) (byteView *ByteView, err error) {
	if key == "" {
		return &ByteView{}, errors.New("key is required")
	}
	if view, ok1 := g.mainCache.get(key); ok1 {
		log.Println("cache hit")
		log.Printf("[now cache] %#v", g.mainCache.cache)
		return view, nil
	}
	return g.load(key)
}

func (g *Group) load(key string) (byteView *ByteView, err error) {
	bytes, err := g.single.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if bytes, err1 := g.getFormPeer(peer, key); err1 == nil {
					log.Println("[get from peer]", bytes.String())
					return bytes, err1
				}
			}
		} else {
			log.Println("[g.peers is nil]")
		}
		return g.getLocally(key)
	})

	if err == nil {
		return bytes.(*ByteView), nil
	}
	return
}

func (g *Group) getLocally(key string) (*ByteView, error) {
	get, err := g.getter.Get(key)
	if err != nil {
		return &ByteView{}, err
	}
	value := &ByteView{byteView: cloneByte(get)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, val *ByteView) {
	//g.mainCache.mu.Lock()
	//defer g.mainCache.mu.Unlock()
	g.mainCache.add(key, val)
}

func (g *Group) getFormPeer(peerGetter PeerGetter, key string) (*ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{
		Value: nil,
	}
	err := peerGetter.Get(req, res)
	if err != nil {
		return &ByteView{}, err
	}

	return &ByteView{byteView: res.Value}, nil
}
