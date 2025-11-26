package internal

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type Logger struct {
	level      string
	accessFile *os.File
}

func (l *Logger) log(level string, format string, args ...any) {
	var levels = map[string]int{"debug": 0, "info": 1, "warn": 2, "error": 3}
	if levels[level] >= levels[l.level] {
		fmt.Printf("\n["+level+"] "+format, args...)
	}
}

func (l *Logger) access(format string, args ...any) {
	if l.accessFile != nil {
		fmt.Fprintf(l.accessFile, format, args...)
	}
}

func (l *Logger) LoggerMiddleware(mux http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		start := time.Now()
		mux.ServeHTTP(rw, r)
		duration := time.Now().Sub(start)
		l.LogRequest(r, duration.Microseconds())
	})
}

func (l *Logger) LogRequest(r *http.Request, duration int64) {
	l.log("debug", "Received request from %s %s %s %s Host: %s User-Agent: %s Accept: %s Duration: %dμs\n",
		r.RemoteAddr, r.Method, r.URL.Path, r.Proto, r.Host, strings.Fields(r.UserAgent())[0], "*/*", duration)
}
