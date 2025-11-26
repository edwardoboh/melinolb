package internal

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type HealthChecker struct {
	backends []*Backend
	config   *Config
	ticker   *time.Ticker
	logger   *Logger
}

func NewHealthChecker(backends []*Backend, cfg *Config, l *Logger) *HealthChecker {
	return &HealthChecker{
		backends: backends,
		config:   cfg,
		logger:   l,
	}
}

func (hc *HealthChecker) Start(ctx context.Context) {
	checkInterval := os.Getenv("HEALTH_CHECK_INTERVAL")
	if checkInterval == "" {
		checkInterval = "10"
	}

	interval, err := strconv.Atoi(checkInterval)
	if err != nil {
		log.Fatal(err)
	}

	hc.ticker = time.NewTicker(time.Duration(interval) * time.Second)

	go func(ticker *time.Ticker) {
		for {
			select {
			case <-ticker.C:
				hc.RunHealthChecks()
			case <-ctx.Done():
				ticker.Stop()
			}
		}
	}(hc.ticker)
}

func (hc *HealthChecker) Stop() {
	hc.ticker.Stop()
}

func (hc *HealthChecker) RunHealthChecks() {
	for _, be := range hc.backends {
		go func(backend *Backend) {
			url := backend.url.String()
			response, err := http.Get(url)

			if err != nil || response.StatusCode != http.StatusOK {
				backend.status.Store(false)
				hc.logger.log("error", "health check failed for backend: %v", url)
				return
			}

			backend.status.Store(true)
			hc.logger.log("debug", "health check is now complete for: %s", url)
		}(be)
	}
}
