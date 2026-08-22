package loadbalancer

import (
	"math/rand"

	"github.com/lb/internal/backend"
)

func (lb *LoadBalancer) chooseBackend() *backend.Backend {

	switch lb.strategy {
	case RoundRobin:
		return lb.roundRobin()
	case PowerOfTwo:
		return lb.powerOfTwo()

	}

	return nil
}

func (lb *LoadBalancer) roundRobin() *backend.Backend {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	n := len(lb.backends)
	if n == 0 {
		return nil
	}

	initialIndex := lb.current

	for i := 0; i < n; i++ {
		idx := (initialIndex + i) % n

		if lb.backends[idx].IsAlive() {
			lb.current = (idx + 1) % n
			return lb.backends[idx]
		}
	}

	return nil
}

func (lb *LoadBalancer) powerOfTwo() *backend.Backend {
	i := rand.Intn(len(lb.backends))
	j := rand.Intn(len(lb.backends))

	for j == i {
		j = rand.Intn(len(lb.backends))
	}

	b1 := lb.backends[i]
	b2 := lb.backends[j]

	if b1.ActiveCount() <= b2.ActiveCount() {
		return b1
	}

	return b2
}
