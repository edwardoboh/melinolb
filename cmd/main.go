package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	lib "github.com/edwardoboh/melinoLB/internal"
)

func testing(res http.ResponseWriter, req *http.Request) {
	lib.LogRequest(req)
	res.Write([]byte("Hello from the function"))
}
func testing2(res http.ResponseWriter, req *http.Request) {
	lib.LogRequest(req)
	res.Write([]byte("Hello from testing 2"))
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", testing)
	mux.HandleFunc("/bat", testing2)

	s := &http.Server{
		Addr:         ":80",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	fmt.Println("Server starting on port 80")
	log.Fatal(s.ListenAndServe())
}
