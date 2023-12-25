package etcd

import (
	"context"
	clientv3 "go.etcd.io/etcd/client/v3"
	"log"
	"testing"
	"time"
)

func TestEtcd(t *testing.T) {
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
	timeout, cancelFunc := context.WithTimeout(context.Background(), time.Second*5)
	defer cancelFunc()
	get, err2 := Client.Get(timeout, "", clientv3.WithPrefix())
	if err2 != nil {
		t.Error(err2)
		return
	}
	for _, kv := range get.Kvs {
		t.Log(string(kv.Key), string(kv.Value))
	}
}
