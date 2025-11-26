package internal

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type Balancer interface {
	Select(r *http.Request, bes []*Backend) (*Backend, error)
}

type RoundRobinBalancer struct {
	current atomic.Int32
}

func (rb *RoundRobinBalancer) Select(r *http.Request, bes []*Backend) (*Backend, error) {
	healthyBackends := make([]*Backend, 0)
	for _, be := range bes {
		if be.status.Load() {
			healthyBackends = append(healthyBackends, be)
		}
	}

	if len(healthyBackends) == 0 {
		return nil, fmt.Errorf("service temporarily unavailable")
	}

	rb.current.Store(rb.current.Add(1) % int32(len(healthyBackends)))
	return healthyBackends[rb.current.Load()], nil
}
