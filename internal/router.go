package internal

import (
	"errors"
	"net/url"
	"slices"
	"sync"
)

var DefaultRouter *Router

// SECTION - ServerPool
type ServerPool struct {
	enabled        []url.URL
	backends       []url.URL
	healthEndpoint string
	current        int
	s              sync.RWMutex
}

func NewServerPool() *ServerPool {
	return &ServerPool{enabled: []url.URL{}, backends: []url.URL{}, healthEndpoint: "/", current: 0}
}

func (pool *ServerPool) NextBackend() url.URL {
	pool.s.Lock()
	defer pool.s.Unlock()
	pool.current = (pool.current + 1) % len(pool.backends)
	return pool.backends[pool.current]
}

func (pool *ServerPool) RegisterBackend(urls []url.URL) {
	pool.s.Lock()
	defer pool.s.Unlock()
	pool.enabled = append(pool.enabled, urls...)
	pool.enabled = slices.Compact(pool.enabled)
	pool.backends = pool.enabled
}

func (pool *ServerPool) Deregister(urls []url.URL) error {
	return nil
}

func (pool *ServerPool) ConfigureHealthCheck(path string) {
	pool.s.Lock()
	defer pool.s.Unlock()
	pool.healthEndpoint = "/"
}

// SECTION - Router
type Router struct {
	routes map[string]*ServerPool
	s      sync.RWMutex
}

func NewRouter() *Router {
	return &Router{routes: make(map[string]*ServerPool)}
}

func (r *Router) FindRoute(path string) (error, url.URL) {
	err, sPool := r.GetServerPool(path)
	if err != nil {
		return err, url.URL{}
	}

	return nil, sPool.NextBackend()
}

func (r *Router) AddRoute(path string, pool *ServerPool) error {
	r.s.Lock()
	defer r.s.Unlock()
	if route := r.routes[path]; route != nil {
		return errors.New("path already exist. append routes instead or use a different name")
	}

	if pool == nil {
		pool = NewServerPool()
	}

	r.routes[path] = pool
	return nil
}

func (r *Router) DeleteRoute(path string) {
	r.s.Lock()
	defer r.s.Unlock()
	delete(r.routes, path)
}

func (r *Router) GetServerPool(path string) (error, *ServerPool) {
	r.s.RLock()
	defer r.s.RUnlock()
	if route := r.routes[path]; route == nil {
		return errors.New("no such route exists"), nil
	}

	return nil, r.routes[path]
}
