package selector

import (
	"hash/fnv"
	"net/http"

	"github.com/lb/internal/backend"
)

type HashSelector struct {
}

func (sl *HashSelector) Choose(backends []*backend.Backend, _ http.ResponseWriter, r *http.Request) *backend.Backend {
	hash := fnv.New64a()
	hash.Write([]byte(r.RemoteAddr))

	idx := hash.Sum64() % uint64(len(backends))

	return backends[idx]
}
