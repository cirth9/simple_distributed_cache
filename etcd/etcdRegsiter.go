package etcd

import (
	clientv3 "go.etcd.io/etcd/client/v3"
	"log"
	"time"
)

var (
	err    error
	Client *clientv3.Client
)

var (
	defaultEndPoints   = []string{"47.115.217.189:2379"}
	defaultDialTimeout = 5 * time.Second
)

type ConfigEtcd struct {
	EndPoints   []string
	DialTimeout time.Duration
	WatcherTime time.Duration
}

func (e *ConfigEtcd) InitDiscovery(endPoints []string, dialTimeout time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("[etcd init] panic")
		}
	}()

	if dialTimeout == 0 {
		dialTimeout = defaultDialTimeout
	}

	if endPoints == nil {
		endPoints = defaultEndPoints
	}

	Client, err = clientv3.New(clientv3.Config{
		Endpoints:   endPoints,
		DialTimeout: dialTimeout,
	})
	if err != nil {
		panic(err)
	}
}
