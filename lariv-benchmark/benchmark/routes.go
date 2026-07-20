package benchmark

import (
	"encoding/json"
	"net/http"

	"github.com/lariv-in/lariv"
	"github.com/lariv-in/lariv/getters"
	"github.com/lariv-in/lariv/registry"
)

func pluginRoutes() lariv.PluginFeatures[lariv.Route] {
	return lariv.PluginFeatures[lariv.Route]{
		Entries: []registry.Pair[string, lariv.Route]{
			{
				Key: "benchmark.ListRoute",
				Value: lariv.Route{
					Path:    "GET /api/articles/",
					Handler: lariv.NewDynamicView("benchmark.ListRouteView"),
				},
			},
			{
				Key: "benchmark.CreateRoute",
				Value: lariv.Route{
					Path:    "POST /api/articles/",
					Handler: lariv.NewDynamicView("benchmark.CreateRouteView"),
				},
			},
			{
				Key: "benchmark.DetailRoute",
				Value: lariv.Route{
					Path:    "GET /api/articles/{id}/",
					Handler: lariv.NewDynamicView("benchmark.DetailRouteView"),
				},
			},
			{
				Key: "benchmark.UpdateRoute",
				Value: lariv.Route{
					Path:    "PUT /api/articles/{id}/",
					Handler: lariv.NewDynamicView("benchmark.UpdateRouteView"),
				},
			},
			{
				Key: "benchmark.DeleteRoute",
				Value: lariv.Route{
					Path:    "DELETE /api/articles/{id}/",
					Handler: lariv.NewDynamicView("benchmark.DeleteRouteView"),
				},
			},
			{
				Key: "benchmark.TruncateRoute",
				Value: lariv.Route{
					Path: "POST /api/truncate/",
					Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						db, err := getters.DBFromContext(r.Context())
						if err != nil {
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}
						if err := db.Exec("TRUNCATE TABLE articles RESTART IDENTITY CASCADE;").Error; err != nil {
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}
						w.WriteHeader(http.StatusNoContent)
					}),
				},
			},
			{
				Key: "benchmark.CounterRoute",
				Value: lariv.Route{
					Path: "POST /api/counter/",
					Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						var req struct {
							Counter int `json:"counter"`
						}
						if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
							http.Error(w, err.Error(), http.StatusBadRequest)
							return
						}
						req.Counter++
						w.Header().Set("Content-Type", "application/json")
						if err := json.NewEncoder(w).Encode(req); err != nil {
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}
					}),
				},
			},
			{
				Key: "benchmark.WebsocketRoute",
				Value: lariv.Route{
					Path:    "GET /api/ws/",
					Handler: http.HandlerFunc(BenchmarkWSHandler),
				},
			},
		},
	}
}
