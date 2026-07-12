package main

import (
	"log"

	"github.com/UniquityVentures/lamu/lamu"
	"github.com/UniquityVentures/lamu/registry"
	"lamu-benchmark/benchmark"
)

func main() {
	plugins := []registry.Pair[string, lamu.Plugin]{
		benchmark.GetPlugin(),
	}

	config, err := lamu.LoadConfigFromFile("config.toml", plugins)
	if err != nil {
		log.Fatalf("failed loading configuration file: %v", err)
	}

	if err := lamu.Start(config, plugins); err != nil {
		log.Fatalf("failed executing application entry: %v", err)
	}
}
