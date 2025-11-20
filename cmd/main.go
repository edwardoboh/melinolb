package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

func init() {
	DefaultRouter = NewRouter()
	serverPool := NewServerPool()
	serverPool.RegisterBackend([]url.URL{
		{Scheme: "http", Host: "localhost:8080", Path: "/"},
		{Scheme: "http", Host: "localhost:8081", Path: "/"},
	})
	DefaultRouter.AddRoute("/", serverPool)
}

func main() {
	// Start health check background job
	StartJob(context.Background())

	// Start main application server
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err, routeUrl := DefaultRouter.FindRoute("/")
		if err != nil {
			fmt.Println(err.Error())
			http.NotFound(w, r)
			return
		}

		revProxy := httputil.NewSingleHostReverseProxy(&routeUrl)
		revProxy.ServeHTTP(w, r)
	})

	s := &http.Server{
		Addr:         ":80",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	fmt.Printf("Load Balancer Listening on Port: %d", PORT)
	log.Fatal(s.ListenAndServe())
}
