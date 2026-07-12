package benchmark

import (
	"github.com/UniquityVentures/lamu/lamu"
	"github.com/UniquityVentures/lamu/registry"
)

func pluginRoutes() lamu.PluginFeatures[lamu.Route] {
	return lamu.PluginFeatures[lamu.Route]{
		Entries: []registry.Pair[string, lamu.Route]{
			{
				Key: "benchmark.ListRoute",
				Value: lamu.Route{
					Path:    "GET /api/articles/",
					Handler: lamu.NewDynamicView("benchmark.ListRouteView"),
				},
			},
			{
				Key: "benchmark.CreateRoute",
				Value: lamu.Route{
					Path:    "POST /api/articles/",
					Handler: lamu.NewDynamicView("benchmark.CreateRouteView"),
				},
			},
			{
				Key: "benchmark.DetailRoute",
				Value: lamu.Route{
					Path:    "GET /api/articles/{id}/",
					Handler: lamu.NewDynamicView("benchmark.DetailRouteView"),
				},
			},
			{
				Key: "benchmark.UpdateRoute",
				Value: lamu.Route{
					Path:    "PUT /api/articles/{id}/",
					Handler: lamu.NewDynamicView("benchmark.UpdateRouteView"),
				},
			},
			{
				Key: "benchmark.DeleteRoute",
				Value: lamu.Route{
					Path:    "DELETE /api/articles/{id}/",
					Handler: lamu.NewDynamicView("benchmark.DeleteRouteView"),
				},
			},
		},
	}
}
