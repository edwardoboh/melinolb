package internal

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

func LoggerMiddleware(mux http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		start := time.Now()
		mux.ServeHTTP(rw, r)
		duration := time.Now().Sub(start)
		LogRequest(r, duration.Microseconds())
	})
}

func LogRequest(r *http.Request, duration int64) {
	fmt.Println("_________________________________________")
	fmt.Printf("Received request from %s\n%s %s %s\nHost: %s\nUser-Agent: %s\nAccept: %s\nDuration: %dμs\n",
		r.RemoteAddr, r.Method, r.URL.Path, r.Proto, r.Host, strings.Fields(r.UserAgent())[0], "*/*", duration)
}
