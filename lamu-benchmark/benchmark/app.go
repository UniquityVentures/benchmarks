package benchmark

import (
	"embed"

	"github.com/lariv-in/lago"
	"github.com/lariv-in/lago/registry"
	"github.com/lariv-in/lago/views"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

func GetPlugin() registry.Pair[string, lago.Plugin] {
	return registry.Pair[string, lago.Plugin]{
		Key: "benchmark",
		Value: lago.Plugin{
			Type:        lago.PluginTypeApp,
			VerboseName: "Benchmark",
			Migrations: lago.PluginStages(func() lago.PluginFeatures[lago.UsefulFilesystem] {
				return lago.PluginFeatures[lago.UsefulFilesystem]{
					Entries: []registry.Pair[string, lago.UsefulFilesystem]{
						{Key: "benchmark", Value: migrationFS},
					},
				}
			}),
			Views: lago.PluginStages(func() lago.PluginFeatures[*views.View] {
				return pluginViews()
			}),
			Routes: lago.PluginStages(func() lago.PluginFeatures[lago.Route] {
				return pluginRoutes()
			}),
			Models: lago.PluginStages(func() lago.PluginFeatures[any] {
				return pluginModels()
			}),
			Layers: lago.PluginStages(func() lago.PluginFeatures[views.GlobalLayer] {
				return lago.PluginFeatures[views.GlobalLayer]{
					Entries: []registry.Pair[string, views.GlobalLayer]{
						{Key: "gzip_layer", Value: GzipLayer{}},
					},
				}
			}),
		},
	}
}
