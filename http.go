package simpleCache

import (
	"context"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/protobuf/proto"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	pb "simpleCache/cacheProtobuf"
	"simpleCache/consistentHash"
	"simpleCache/etcd"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const defaultBasePath = "/_cache/"

// HTTPPool implements PeerPicker for a pool of HTTP peers.
type HTTPPool struct {
	// this peer's base URL, e.g. "https://example.net:8000"
	self        string
	basePath    string
	mu          sync.Mutex
	peers       *consistentHash.ConsistentHash
	httpGetters map[string]*HttpGetter
}

var (
	Pool    *HTTPPool
	keyName string
)

func NewHTTPPool(self string) {
	Pool = &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
		peers:    consistentHash.NewConsistentHash(consistentHash.DefaultReplicas, nil),
	}

	//todo 从etcd获取分布式结点，此时etcd中还不包含self
	timeout, cancelFunc := context.WithTimeout(context.Background(), time.Second*5)
	defer cancelFunc()
	get, err := etcd.Client.Get(timeout, "Cache&", clientv3.WithPrefix())
	if err != nil {
		log.Println("[NewHttpPool] etcd get error", err)
		panic(err)
	}

	//todo 将self加入etcd
	keyName = "Cache&" + strconv.Itoa(int(time.Now().Unix())) + strconv.Itoa(rand.Intn(1000000))
	timeout1, cancelFunc1 := context.WithTimeout(context.Background(), time.Second*5)
	defer cancelFunc1()
	if _, err = etcd.Client.Put(timeout1, keyName, self); err != nil {
		log.Println(err)
	}

	//todo 将各种结点接入Pool内
	Pool.Set(self)
	for _, kv := range get.Kvs {
		Pool.Set(string(kv.Value))
	}

	go func() {

		defer log.Println("[etcd Watcher] finished!")
		for {
			ctx := context.TODO()
			log.Println("watch")

			//todo 设置要监听的键前缀
			prefix := "Cache&"

			//todo 创建一个 Watcher
			watcher := etcd.Client.Watch(ctx, prefix, clientv3.WithPrefix())

			//todo 循环监听事件
			for resp := range watcher {
				for _, event := range resp.Events {
					switch event.Type {
					case clientv3.EventTypePut:
						log.Println("watch put", event.Kv)
						Pool.Set(string(event.Kv.Value))
					case clientv3.EventTypeDelete:
						log.Println("watch delete", event.Kv)
						Pool.peers.Del(string(event.Kv.Key))
					}
				}
			}
			time.Sleep(time.Second * 3)
		}
	}()
}

func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*HttpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &HttpGetter{baseURL: peer + p.basePath}
	}
}

func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	log.Println("[HTTPPool] PickPeer")
	log.Printf("%#v", p.peers)
	p.mu.Lock()
	defer p.mu.Unlock()
	if get := p.peers.Get(key); get != "" && get != p.self {
		p.Log("Pick peer %s", get)
		return p.httpGetters[get], true
	}
	p.Log("can't Pick anything,the get is %s ", p.peers.Get(key))
	return nil, false
}

func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//todo 浏览器访问的话，还会额外发出一个url路径为/favicon.ico，导致panic，这里后面的||判断条件主要是为了避免panic的
	if strings.HasPrefix(r.URL.Path, "/favicon.ico") {
		return
	}

	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// /<basepath>/<groupname>/<key> required
	// default base path is _cache
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	protoRes, err := proto.Marshal(&pb.Response{
		Value: view.ByteSlice(),
	})
	w.Header().Set("Content-Type", "application/octet-stream")
	_, err = w.Write(protoRes)
	if err != nil {
		panic(err)
	}
}

type HttpGetter struct {
	baseURL string
}

func (h *HttpGetter) Get(req *pb.Request, res *pb.Response) error {
	getUrl := fmt.Sprintf("http://%v%v/%v",
		h.baseURL,
		url.QueryEscape(req.GetGroup()),
		url.QueryEscape(req.GetKey()))
	response, err := http.Get(getUrl)
	log.Println("[GET]", getUrl)
	if err != nil {
		log.Println(err)
		return err
	}
	defer func(Body io.ReadCloser) {
		err1 := Body.Close()
		if err1 != nil {
			panic(err1)
		}
	}(response.Body)

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", response.Status)
	}

	bytes, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("server returned: %v", response.Status)
	}

	if err = proto.Unmarshal(bytes, res); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}
	log.Println("[GET]", res)
	return nil
}

// Start todo 启动结点服务
func Start(address string) {
	go func() {
		log.Println(http.ListenAndServe(address, Pool))
	}()
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
	log.Println("cache server stop success...")
	timeout, cancelFunc := context.WithTimeout(context.Background(), time.Second*5)
	defer cancelFunc()
	if _, err := etcd.Client.Delete(timeout, keyName); err != nil {
		log.Println(err)
		panic(err)
	}
}

var _ PeerGetter = (*HttpGetter)(nil)
