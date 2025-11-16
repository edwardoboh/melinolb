package internal

import (
	"fmt"
	"net/http"
	"strings"
)

func LogRequest(r *http.Request) {
	fmt.Println("_________________________________________")
	fmt.Printf("Received request from %s\n%s %s %s\nHost: %s\nUser-Agent: %s\nAccept: %s\n",
		r.RemoteAddr, r.Method, r.URL.Path, r.Proto, r.Host, strings.Fields(r.UserAgent())[0], "*/*")
}
