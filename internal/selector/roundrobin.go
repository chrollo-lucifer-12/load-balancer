package selector

import (
	"net/http"
	"sync"

	"github.com/lb/internal/backend"
)

type RoundRobinSelector struct {
	mu      sync.Mutex
	current int
}

func (sl *RoundRobinSelector) Choose(backends []*backend.Backend, _ http.ResponseWriter, _ *http.Request) *backend.Backend {
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
