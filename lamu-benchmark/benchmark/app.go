package benchmark

import (
	"embed"

	"github.com/UniquityVentures/lamu/lamu"
	"github.com/UniquityVentures/lamu/registry"
	"github.com/UniquityVentures/lamu/views"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

func GetPlugin() registry.Pair[string, lamu.Plugin] {
	return registry.Pair[string, lamu.Plugin]{
		Key: "benchmark",
		Value: lamu.Plugin{
			Type:        lamu.PluginTypeApp,
			VerboseName: "Benchmark",
			Migrations: lamu.PluginStages(func() lamu.PluginFeatures[lamu.UsefulFilesystem] {
				return lamu.PluginFeatures[lamu.UsefulFilesystem]{
					Entries: []registry.Pair[string, lamu.UsefulFilesystem]{
						{Key: "benchmark", Value: migrationFS},
					},
				}
			}),
			Views: lamu.PluginStages(func() lamu.PluginFeatures[*views.View] {
				return pluginViews()
			}),
			Routes: lamu.PluginStages(func() lamu.PluginFeatures[lamu.Route] {
				return pluginRoutes()
			}),
			Models: lamu.PluginStages(func() lamu.PluginFeatures[any] {
				return pluginModels()
			}),
			Layers: lamu.PluginStages(func() lamu.PluginFeatures[views.GlobalLayer] {
				return lamu.PluginFeatures[views.GlobalLayer]{
					Entries: []registry.Pair[string, views.GlobalLayer]{
						{Key: "gzip_layer", Value: GzipLayer{}},
					},
				}
			}),
		},
	}
}
