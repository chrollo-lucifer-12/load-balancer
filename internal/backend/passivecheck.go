package backend

import (
	"net/http"
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

func (h *PassiveHealthHandler) RecordResponse(status int) {
	if status >= 500 {
		h.backend.RecordFailure(h.maxFailCount)
		return
	}

	h.backend.RecordSuccess()
}
