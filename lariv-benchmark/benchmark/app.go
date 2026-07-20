package benchmark

import (
	"embed"

	"github.com/lariv-in/lariv"
	"github.com/lariv-in/lariv/registry"
	"github.com/lariv-in/lariv/views"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

func GetPlugin() registry.Pair[string, lariv.Plugin] {
	return registry.Pair[string, lariv.Plugin]{
		Key: "benchmark",
		Value: lariv.Plugin{
			Type:        lariv.PluginTypeApp,
			VerboseName: "Benchmark",
			Migrations: lariv.PluginStages(func() lariv.PluginFeatures[lariv.UsefulFilesystem] {
				return lariv.PluginFeatures[lariv.UsefulFilesystem]{
					Entries: []registry.Pair[string, lariv.UsefulFilesystem]{
						{Key: "benchmark", Value: migrationFS},
					},
				}
			}),
			Views: lariv.PluginStages(func() lariv.PluginFeatures[*views.View] {
				return pluginViews()
			}),
			Routes: lariv.PluginStages(func() lariv.PluginFeatures[lariv.Route] {
				return pluginRoutes()
			}),
			Models: lariv.PluginStages(func() lariv.PluginFeatures[any] {
				return pluginModels()
			}),
			Layers: lariv.PluginStages(func() lariv.PluginFeatures[views.GlobalLayer] {
				return lariv.PluginFeatures[views.GlobalLayer]{
					Entries: []registry.Pair[string, views.GlobalLayer]{
						{Key: "gzip_layer", Value: GzipLayer{}},
					},
				}
			}),
		},
	}
}
