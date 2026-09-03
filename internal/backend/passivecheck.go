package backend

import (
	"net/http"

	"github.com/lb/internal/rw"
)

type PassiveHealthHandler struct {
	backend      *Backend
	next         http.Handler
	maxFailCount int64
}

func NewPassiveHealthHandlder(backend *Backend,
	next http.Handler,
	maxFailCount int64) *PassiveHealthHandler {

	return &PassiveHealthHandler{
		backend:      backend,
		next:         next,
		maxFailCount: maxFailCount,
	}
}

func (h *PassiveHealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rec := w.(*rw.ResponseWrapper)

	h.next.ServeHTTP(rec, r)

	if rec.Status >= 500 {

		h.backend.RecordFailure(h.maxFailCount)
		return
	}

	h.backend.RecordSuccess()
}
