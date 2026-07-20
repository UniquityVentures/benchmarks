package main

import (
	"lariv-benchmark/benchmark"
	"log"
	"runtime"

	"github.com/lariv-in/lariv"
	"github.com/lariv-in/lariv/registry"
)

func main() {
	runtime.GOMAXPROCS(4)
	plugins := []registry.Pair[string, lariv.Plugin]{
		benchmark.GetPlugin(),
	}

	config, err := lariv.LoadConfigFromFile("config.toml", plugins)
	if err != nil {
		log.Fatalf("failed loading configuration file: %v", err)
	}

	if err := lariv.Start(config, plugins); err != nil {
		log.Fatalf("failed executing application entry: %v", err)
	}
}
