package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	lib "github.com/edwardoboh/melinolb/internal"
)

func init() {
	lib.DefaultRouter = lib.NewRouter()
	serverPool := lib.NewServerPool()
	serverPool.RegisterBackend([]url.URL{
		{Scheme: "http", Host: "localhost:8080", Path: "/"},
		{Scheme: "http", Host: "localhost:8081", Path: "/"},
	})
	lib.DefaultRouter.AddRoute("/", serverPool)
}

func main() {
	// Start health check background job
	lib.StartJob(context.Background())

	// Start main application server
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err, routeUrl := lib.DefaultRouter.FindRoute("/")
		if err != nil {
			fmt.Println(err.Error())
			http.NotFound(w, r)
			return
		}

		revProxy := httputil.NewSingleHostReverseProxy(&routeUrl)
		revProxy.ServeHTTP(w, r)
	})

	// Add logging middleware
	loggedMux := lib.LoggerMiddleware(mux)

	s := &http.Server{
		Addr:         ":80",
		Handler:      loggedMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	fmt.Printf("Load Balancer Listening on Port: %d\n", 80)
	log.Fatal(s.ListenAndServe())
}
