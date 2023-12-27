package main

import (
	"errors"
	"flag"
	"fmt"
	cache "github.com/thewisecirno/simple_distributed_cache"
	"github.com/thewisecirno/simple_distributed_cache/etcd"
	"log"
)

var db = map[string]string{
	"A":      "1",
	"B":      "2",
	"C":      "3",
	"D":      "4",
	"E":      "5",
	"F":      "6",
	"G":      "7",
	"H":      "8",
	"I":      "9",
	"J":      "10",
	"K":      "11",
	"Cirno":  "123",
	"Koish":  "421",
	"Satori": "353",
	"Joker":  "123",
}

func main() {
	addr := flag.String("addr", "", "ip:port")
	//data := flag.String("kv", " ", "db data")
	flag.Parse()

	//log.Println(*addr, *data)
	//kvData := strings.Split(*data, " ")
	//for _, kv := range kvData {
	//	kvDataSlice := strings.Split(kv, ":")
	//	db[kvDataSlice[0]] = kvDataSlice[1]
	//}
	//log.Println(db)

	if etcd.Client == nil {
		panic(errors.New("etcd client  is  nil!!"))
	}
	cache.NewHTTPPool(*addr)
	cache.NewGroup("scores", 2<<10, cache.GetterHandler(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
	log.Println("_cache is running at", *addr)
	cache.Start(*addr)
}
