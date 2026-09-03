package selector

import (
	"net/http"

	"github.com/lb/internal/backend"
)

type SelectorType string

const (
	RoundRobin        SelectorType = "round_robin"
	PowerOfTwo        SelectorType = "power_of_two"
	ConsistentHashing SelectorType = "consistent_hashing"
	Sticky            SelectorType = "sticky"
)

func NewSelector(sType SelectorType) Selector {
	switch sType {
	case RoundRobin:
		return &RoundRobinSelector{}
	case PowerOfTwo:
		return &PowerOfTwoSelector{}
	case ConsistentHashing:
		return NewConsistentHashSelector(3)
	default:
		return nil
	}
}

type Selector interface {
	Choose(backends []*backend.Backend, w http.ResponseWriter, r *http.Request) *backend.Backend
}
