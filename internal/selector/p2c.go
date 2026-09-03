package selector

import (
	"math/rand"
	"net/http"
	"sync"

	"github.com/lb/internal/backend"
)

type PowerOfTwoSelector struct {
	mu sync.Mutex
}

func (sl *PowerOfTwoSelector) Choose(backends []*backend.Backend, _ http.ResponseWriter, _ *http.Request) *backend.Backend {
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
