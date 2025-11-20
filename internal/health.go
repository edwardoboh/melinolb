package internal

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

func StartJob(ctx context.Context) {
	checkInterval := os.Getenv("HEALTH_CHECK_INTERVAL")
	if checkInterval == "" {
		checkInterval = "10"
	}

	interval, err := strconv.Atoi(checkInterval)
	if err != nil {
		log.Fatal(err)
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)

	go func(ticker *time.Ticker) {
		for {
			select {
			case <-ticker.C:
				RunHealthChecks()
			case <-ctx.Done():
				ticker.Stop()
			}
		}
	}(ticker)
}

func RunHealthChecks() {
	DefaultRouter.s.RLock()
	defer DefaultRouter.s.RUnlock()

	for path, pool := range DefaultRouter.routes {
		go func(path string, pool *ServerPool) {
			var healthyMap = make(map[url.URL]bool)

			for _, url := range pool.enabled {
				endpoint := fmt.Sprintf("%s://%s/%s", url.Scheme, url.Host, pool.healthEndpoint)
				response, err := http.Get(endpoint)

				if err != nil || response.StatusCode != http.StatusOK {
					healthyMap[url] = false
					fmt.Printf("\nhealth check failed for backend: %v", url)
					continue
				}

				healthyMap[url] = true
			}

			var healthyBackends = []url.URL{}
			for url, isHealthy := range healthyMap {
				if isHealthy {
					healthyBackends = append(healthyBackends, url)
				}
			}

			pool.s.Lock()
			defer pool.s.Unlock()
			pool.backends = healthyBackends
			fmt.Printf("\nhealth check is now complete for: %s", path)
		}(path, pool)
	}
}
