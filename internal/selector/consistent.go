package selector

import (
	"hash/fnv"
	"net/http"
	"sort"
	"strconv"

	"github.com/lb/internal/backend"
)

type ConsistentHashSelector struct {
	hashes []uint32
	nodes  map[uint32]*backend.Backend

	virtualNodes int
}

func NewConsistentHashSelector(virtualNodes int) *ConsistentHashSelector {
	return &ConsistentHashSelector{
		nodes:        map[uint32]*backend.Backend{},
		virtualNodes: virtualNodes,
	}
}

func (sl *ConsistentHashSelector) Update(backends []*backend.Backend) {
	hashes := make([]uint32, 0, len(backends)*sl.virtualNodes)
	nodes := make(map[uint32]*backend.Backend)

	for _, b := range backends {
		if !b.IsAlive() {
			continue
		}

		for i := 0; i < sl.virtualNodes; i++ {
			hash := hashKey(
				b.URL.String() + "#" + strconv.Itoa(i),
			)

			hashes = append(hashes, hash)
			nodes[hash] = b
		}
	}

	sort.Slice(hashes, func(i, j int) bool {
		return hashes[i] < hashes[j]
	})

	sl.hashes = hashes
	sl.nodes = nodes
}

func (sl *ConsistentHashSelector) Choose(backends []*backend.Backend, _ http.ResponseWriter, r *http.Request) *backend.Backend {
	if len(sl.hashes) == 0 {
		return nil
	}

	hash := hashKey(r.RemoteAddr)

	idx := sort.Search(len(sl.hashes), func(i int) bool { return sl.hashes[i] >= hash })

	if idx == len(sl.hashes) {
		idx = 0
	}

	return sl.nodes[sl.hashes[idx]]
}

func hashKey(key string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return h.Sum32()
}
