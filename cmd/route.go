package main

import (
	"errors"
	"net/url"
	"slices"
	"sync"
)

const PORT = 80

var (
	DefaultRouter *Router
)

func init() {
	DefaultRouter = NewRouter()
	DefaultRouter.AddRoute("/", []url.URL{})
}

// SECTION - ServerPool
type ServerPool struct {
	backends []url.URL
	current  int
	s        sync.RWMutex
}

func NewServePool() *ServerPool {
	return &ServerPool{backends: []url.URL{}, current: 0}
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
	pool.backends = append(pool.backends, urls...)
	pool.backends = slices.Compact(pool.backends)
}

func (pool *ServerPool) Deregister(urls []url.URL) error {
	return nil
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

func (r *Router) AddRoute(path string, paths []url.URL) error {
	r.s.Lock()
	defer r.s.Unlock()
	if route := r.routes[path]; route != nil {
		return errors.New("path already exist. append routes instead or use a different name")
	}

	newPool := NewServePool()
	newPool.RegisterBackend(paths)
	r.routes[path] = NewServePool()
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
