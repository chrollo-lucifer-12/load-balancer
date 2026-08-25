package selector

import (
	"hash/fnv"
	"math/rand"
	"sort"
	"strconv"
	"sync"

	"github.com/lb/internal/backend"
)

type SelectorType string

const (
	RoundRobin SelectorType = "round_robin"
	PowerOfTwo SelectorType = "power_of_two"
)

func NewSelector(sType SelectorType) Selector {
	switch sType {
	case RoundRobin:
		return &RoundRobinSelector{}
	case PowerOfTwo:
		return &PowerOfTwoSelector{}
	default:
		return nil
	}
}

type Selector interface {
	Choose(backends []*backend.Backend, key string) *backend.Backend
}

type RoundRobinSelector struct {
	mu      sync.Mutex
	current int
}

type PowerOfTwoSelector struct {
	mu sync.Mutex
}

type HashSelector struct {
}

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

func (sl *RoundRobinSelector) Choose(backends []*backend.Backend, key string) *backend.Backend {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	n := len(backends)
	if n == 0 {
		return nil
	}

	initialIndex := sl.current

	for i := 0; i < n; i++ {
		idx := (initialIndex + i) % n

		if backends[idx].CanPass() {
			sl.current = (idx + 1) % n
			return backends[idx]
		}
	}

	return nil
}

func (sl *PowerOfTwoSelector) Choose(backends []*backend.Backend, key string) *backend.Backend {
	i := rand.Intn(len(backends))
	j := rand.Intn(len(backends))

	for j == i {
		j = rand.Intn(len(backends))
	}

	b1 := backends[i]
	b2 := backends[j]

	if b1.ActiveCount() <= b2.ActiveCount() {
		return b1
	}

	return b2
}

func (sl *HashSelector) Choose(backends []*backend.Backend, key string) *backend.Backend {
	hash := fnv.New64a()
	hash.Write([]byte(key))

	idx := hash.Sum64() % uint64(len(backends))

	return backends[idx]
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

func (sl *ConsistentHashSelector) Choose(backends []*backend.Backend, key string) *backend.Backend {
	if len(sl.hashes) == 0 {
		return nil
	}

	hash := hashKey(key)

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
