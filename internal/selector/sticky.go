package selector

import (
	"net/http"
	"sync"

	"github.com/lb/internal/backend"
)

type StickySelector struct {
	mu       sync.RWMutex
	sessions map[string]*backend.Backend

	fallback Selector
}

func NewStickySelector() *StickySelector {
	return &StickySelector{
		sessions: make(map[string]*backend.Backend),
		fallback: &RoundRobinSelector{},
	}
}

func (sl *StickySelector) Choose(backends []*backend.Backend, w http.ResponseWriter, r *http.Request) *backend.Backend {

	key := getKey(r)

	if key != "" {
		sl.mu.RLock()
		b := sl.sessions[key]
		sl.mu.RUnlock()

		if b != nil && b.IsAlive() {
			return b
		}
	}

	b := sl.fallback.Choose(backends, w, r)

	if b == nil {
		return nil
	}

	sl.mu.Lock()
	sl.sessions[key] = b
	sl.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "LB_SESSION",
		Value:    key,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	return b
}

func (sl *StickySelector) setSession(key string, b *backend.Backend) {
	sl.mu.Lock()
	sl.sessions[key] = b
	sl.mu.Unlock()
}

func getKey(r *http.Request) string {
	session, err := r.Cookie("LB_SESSION")
	if err != nil {
		return ""
	}

	return session.Value
}
