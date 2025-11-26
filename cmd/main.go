package main

import (
	"log"
	"os"

	lib "github.com/edwardoboh/melinolb/internal"
)

func main() {
	// Load config
	configFile := "config.yaml"
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}

	conf, err := lib.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("could not load configuration file at: %s", configFile)
	}

	// Create Load Balancer
	lb, err := lib.NewLoadBalancer(conf)
	if err != nil {
		log.Fatalf("could not start load balancer: %v", err)
	}

	if err := lb.Start(); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}
