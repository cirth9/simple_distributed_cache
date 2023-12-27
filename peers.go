package simpleCache

import (
	pb "github.com/thewisecirno/simple_distributed_cache/cacheProtobuf"
)

// PeerPicker 抽象出一个key对应一个peerGetter
type PeerPicker interface {
	PickPeer(key string) (PeerGetter, bool)
}

// PeerGetter 通过group_name和key获取到实际对应的值
type PeerGetter interface {
	Get(request *pb.Request, response *pb.Response) error
	//Get(group, key string) ([]byte, error)
}
