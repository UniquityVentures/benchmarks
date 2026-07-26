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

	app, err := lariv.NewBuilder().AddPlugins(plugins).LoadConfigFromFile("config.toml")
	if err != nil {
		log.Fatalf("failed loading configuration file: %v", err)
	}

	if err := app.Start(); err != nil {
		log.Fatalf("failed executing application entry: %v", err)
	}
}
