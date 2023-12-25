package etcd

import (
	clientv3 "go.etcd.io/etcd/client/v3"
	"log"
	"time"
)

var (
	endPoints = []string{"47.115.217.189:2379"}
	err       error
	Client    *clientv3.Client
)

func init() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("[etcd init] panic")
		}
	}()
	Client, err = clientv3.New(clientv3.Config{
		Endpoints:   endPoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		panic(err)
	}
}
