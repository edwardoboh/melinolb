package internal

import (
	"net/http/httputil"
	"net/url"
	"sync/atomic"
)

type RouteHandler struct {
	Route    Route
	Backends []*Backend
	Health   *HealthChecker
	Limiter  *RateLimiter
	Balancer Balancer
}
type Backend struct {
	url    *url.URL
	status atomic.Bool
	proxy  *httputil.ReverseProxy
}
