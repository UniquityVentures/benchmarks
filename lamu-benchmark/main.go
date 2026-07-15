package main

import (
	"log"
	"runtime"

	"github.com/lariv-in/lago"
	"github.com/lariv-in/lago/registry"
	"lago-benchmark/benchmark"
)

func main() {
	runtime.GOMAXPROCS(4)
	plugins := []registry.Pair[string, lago.Plugin]{
		benchmark.GetPlugin(),
	}

	config, err := lago.LoadConfigFromFile("config.toml", plugins)
	if err != nil {
		log.Fatalf("failed loading configuration file: %v", err)
	}

	if err := lago.Start(config, plugins); err != nil {
		log.Fatalf("failed executing application entry: %v", err)
	}
}
