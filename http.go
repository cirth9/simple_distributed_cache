package simpleCache

import (
	"fmt"
	"google.golang.org/protobuf/proto"
	"io"
	"log"
	"net/http"
	"net/url"
	pb "simpleCache/cacheProtobuf"
	"simpleCache/consistentHash"
	"strings"
	"sync"
)

const defaultBasePath = "/_geecache/"

// HTTPPool implements PeerPicker for a pool of HTTP peers.
type HTTPPool struct {
	// this peer's base URL, e.g. "https://example.net:8000"
	self        string
	basePath    string
	mu          sync.Mutex
	peers       *consistentHash.ConsistentHash
	httpGetters map[string]*HttpGetter
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistentHash.NewConsistentHash(consistentHash.DefaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*HttpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &HttpGetter{baseURL: peer + p.basePath}
	}
}

func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if get := p.peers.Get(key); get != "" && get != p.self {
		p.Log("Pick peer %s", get)
		return p.httpGetters[get], true
	}
	return nil, false
}

func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// /<basepath>/<groupname>/<key> required
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
	getUrl := fmt.Sprintf("%v%v/%v",
		h.baseURL,
		url.QueryEscape(req.GetGroup()),
		url.QueryEscape(req.GetKey()))
	response, err := http.Get(getUrl)
	if err != nil {
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

	return nil
}

var _ PeerGetter = (*HttpGetter)(nil)
