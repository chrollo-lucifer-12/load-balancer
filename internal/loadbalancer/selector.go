package loadbalancer

import "github.com/lb/internal/backend"

func (lb *LoadBalancer) chooseBackend() *backend.Backend {

	switch lb.strategy {
	case RoundRobin:
		return lb.roundRobin()
	}

	return nil
}

func (lb *LoadBalancer) roundRobin() *backend.Backend {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	initialIndex := lb.current

	for i := 0; i < len(lb.backends); i++ {
		idx := (initialIndex + i) % len(lb.backends)
		if lb.backends[idx].IsAlive() {
			lb.current = idx
			return lb.backends[idx]
		}
	}

	return nil
}
