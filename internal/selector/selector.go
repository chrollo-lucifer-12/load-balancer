package selector

import (
	"math/rand"
	"sync"

	"github.com/lb/internal/backend"
)

type Selector interface {
	Choose(backends []*backend.Backend) *backend.Backend
}

type RoundRobinSelector struct {
	mu      sync.Mutex
	current int
}

type PowerOfTwoSelector struct {
	mu sync.Mutex
}

func (sl *RoundRobinSelector) Choose(backends []*backend.Backend) *backend.Backend {
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

func (sl *PowerOfTwoSelector) Choose(backends []*backend.Backend) *backend.Backend {
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
