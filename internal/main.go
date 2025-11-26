package internal

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

type LoadBalancer struct {
	Config      *Config
	Logger      *Logger
	Routes      []*RouteHandler
	adminServer *http.Server
}

func NewLoadBalancer(config *Config) (*LoadBalancer, error) {
	logger := Logger{level: config.Logging.Level}
	accessLog := config.Logging.AccessLog
	if accessLog != "-" && accessLog != "" {
		af, err := os.OpenFile(accessLog, 0, 641)
		if err != nil {
			return nil, fmt.Errorf("unable to access access log file: %w", err)
		}
		logger.accessFile = af
	}

	lb := &LoadBalancer{Config: config, Logger: &logger}
	var routes []*RouteHandler

	for _, route := range config.Routes {
		rh, err := lb.createRouteHandlers(route)
		if err != nil {
			return nil, fmt.Errorf("could not create route handle for route %s: %w", route.Id, err)
		}
		routes = append(routes, rh)
	}

	lb.Routes = routes
	return lb, nil
}

func (l *LoadBalancer) createRouteHandlers(route Route) (*RouteHandler, error) {
	rh := &RouteHandler{Route: route}
	var configBackends = make([]string, 0)

	// Create URL list for all backends under route
	switch v := route.Backend.(type) {
	case []interface{}:
		for _, itm := range v {
			if _, ok := l.Config.Backends[itm.(string)]; ok {
				configBackends = append(configBackends, l.Config.Backends[itm.(string)]...)
				continue
			}
			configBackends = append(configBackends, itm.(string))
		}
	case string:
		if _, ok := l.Config.Backends[v]; ok {
			configBackends = append(configBackends, l.Config.Backends[v]...)
		} else {
			configBackends = append(configBackends, v)
		}
	}

	if len(configBackends) == 0 {
		return nil, fmt.Errorf("no backend(s) configured for route: %s", route.Id)
	}

	for _, u := range configBackends {
		url, err := url.Parse(u)
		if err != nil {
			l.Logger.log("warn", "invalid backend route configured for service %s: %s --> %%s")
		}

		backend := Backend{url: url}
		if route.Health != nil && route.Health.Path == "" {
			backend.status.Store(false)
		}

		backend.proxy = httputil.NewSingleHostReverseProxy(url)

		timeout, err := time.ParseDuration(l.Config.Defaults.BackendConnectTimeout)
		if err != nil {
			return nil, fmt.Errorf("invalid `backend_connection_timeout` configuration: %w", err)
		}

		keepAlive, err := time.ParseDuration(l.Config.Defaults.ConnectionKeepAlive)
		if err != nil {
			return nil, fmt.Errorf("invalid `connection_keep_alive` configuration: %w", err)
		}

		backend.proxy.Transport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   timeout,
				KeepAlive: keepAlive,
			}).DialContext,
			MaxIdleConns:        100,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		}
		backend.proxy.ErrorHandler = func(rw http.ResponseWriter, r *http.Request, e error) {
			fmt.Printf("proxy error for backend %s: %v", url.String(), e)
			http.Error(rw, "internal server error", http.StatusInternalServerError)
		}

		rh.Backends = append(rh.Backends, &backend)
	}

	switch route.LB {
	case "round-robin":
		fallthrough
	default:
		rh.Balancer = &RoundRobinBalancer{}
	}

	// TODO - Select Health Checker
	if route.Health != nil {
		rh.Health = NewHealthChecker(rh.Backends, l.Config, l.Logger)
		rh.Health.Start(context.Background())
	} else {
		for _, be := range rh.Backends {
			be.status.Store(true)
		}
	}

	// TODO - Select Rate Limiter

	return rh, nil
}

func (l *LoadBalancer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var rh *RouteHandler

	for _, route := range l.Routes {
		if l.matchRoute(route.Route, r) {
			rh = route
			break
		}
	}

	if rh == nil {
		l.Logger.access("%s %s %d %s", r.Method, r.URL.Path, 404, time.Since(start))
		http.Error(rw, "route not found", 404)
		return
	}

	// TODO - Rate Limit

	// Select backend
	be, err := rh.Balancer.Select(r, rh.Backends)
	if err != nil {
		l.Logger.access("%s %s %d %s", r.Method, r.URL.Path, 500, time.Since(start))
		http.Error(rw, "internal server error", 500)
		return
	}

	// handle sticky-icky
	if rh.Route.Sticky != nil && rh.Route.Sticky.Enabled {
		http.SetCookie(rw, &http.Cookie{
			Name:    rh.Route.Sticky.CookieName,
			Value:   be.url.Host,
			Expires: time.Unix(int64(rh.Route.Sticky.TTL), 0),
		})
	}

	l.Logger.LogRequest(r, int64(time.Since(start)))
	be.proxy.ServeHTTP(rw, r)

	l.Logger.access("%s %s %d %s %s", r.Method, r.URL.Path, 200, time.Since(start), be.url.Host)

}

func (l *LoadBalancer) matchRoute(route Route, r *http.Request) bool {
	if route.Match.Host != "" && route.Match.Host != r.Host {
		return false
	}

	if route.Match.Path != "" {
		if route.Match.Path == "/" {
			return true
		}
		if !strings.HasPrefix(r.URL.Path, route.Match.Path) {
			return false
		}
	}

	if len(route.Match.Methods) > 0 {
		var isMatch = false
		for _, m := range route.Match.Methods {
			if r.Method == m {
				isMatch = true
				break
			}
		}
		if !isMatch {
			return false
		}
	}

	return true
}

func (l *LoadBalancer) setupAdminServer() {
	// TODO - Implement admin server in subroutine
}

func (l *LoadBalancer) Start() error {
	// TODO - Check and set up connection to Admin server

	rdTimeout, err := time.ParseDuration(l.Config.Defaults.ReadTimeout)
	if err != nil {
		l.Logger.log("error", "Default ReadTimeout is invalid: %w", err)
		return fmt.Errorf("Default ReadTimeout is invalid: %w", err)
	}

	wrTimeout, err := time.ParseDuration(l.Config.Defaults.WriteTimeout)
	if err != nil {
		l.Logger.log("error", "Default WriteTimeout is invalid: %w", err)
		return fmt.Errorf("Default WriteTimeout is invalid: %w", err)
	}

	srv := &http.Server{
		Addr:         l.Config.Service.Listen,
		Handler:      l,
		ReadTimeout:  rdTimeout,
		WriteTimeout: wrTimeout,
	}

	l.Logger.log("info", "server running on port: %s", srv.Addr)

	if l.Config.TLS != nil && l.Config.TLS.Enabled {
		return srv.ListenAndServeTLS(l.Config.TLS.CertPath, l.Config.TLS.KeyPath)
	}
	return srv.ListenAndServe()
}

func (l *LoadBalancer) Shutdown(c context.Context) error {
	// TODO - stop health check jobs
	for _, r := range l.Routes {
		r.Health.Stop()
	}

	// TODO - stop rate limiting jobs

	// Shutdown monitoring/admin server
	l.adminServer.Shutdown(c)

	return nil
}
