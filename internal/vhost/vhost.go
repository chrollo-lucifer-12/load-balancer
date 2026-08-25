package vhost

import (
	"fmt"

	"time"

	"github.com/lb/internal/backend"
	"github.com/lb/internal/config"
	"github.com/lb/internal/selector"
)

type VHost struct {
	backends            []*backend.Backend
	sl                  selector.Selector
	healthCheckInterval time.Duration
}

func NewVHost(vhostConfig config.VirtualHost) *VHost {

	sl := selector.NewSelector(selector.SelectorType(vhostConfig.Strategy))
	maxFailCount := vhostConfig.HealthCheck.MaxFailures
	healthCheckInterval := vhostConfig.HealthCheck.Interval

	vh := &VHost{
		sl:                  sl,
		healthCheckInterval: time.Duration(healthCheckInterval) * time.Second,
	}

	backends := make([]*backend.Backend, len(vhostConfig.Backends))

	for i, backendConfig := range vhostConfig.Backends {
		backend, err := backend.NewBackend(backendConfig.URL, backendConfig.Weight, int64(maxFailCount))
		if err != nil {
			fmt.Errorf("new backend err :%w", err)
			return nil
		}

		backends[i] = backend
	}

	vh.backends = backends

	go vh.healthCheck()

	return vh
}

func (vh *VHost) Choose(key string) *backend.Backend {
	return vh.sl.Choose(vh.backends, key)
}
