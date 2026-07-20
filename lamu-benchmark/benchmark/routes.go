package benchmark

import (
	"encoding/json"
	"net/http"

	"github.com/lariv-in/lago"
	"github.com/lariv-in/lago/getters"
	"github.com/lariv-in/lago/registry"
)

func pluginRoutes() lago.PluginFeatures[lago.Route] {
	return lago.PluginFeatures[lago.Route]{
		Entries: []registry.Pair[string, lago.Route]{
			{
				Key: "benchmark.ListRoute",
				Value: lago.Route{
					Path:    "GET /api/articles/",
					Handler: lago.NewDynamicView("benchmark.ListRouteView"),
				},
			},
			{
				Key: "benchmark.CreateRoute",
				Value: lago.Route{
					Path:    "POST /api/articles/",
					Handler: lago.NewDynamicView("benchmark.CreateRouteView"),
				},
			},
			{
				Key: "benchmark.DetailRoute",
				Value: lago.Route{
					Path:    "GET /api/articles/{id}/",
					Handler: lago.NewDynamicView("benchmark.DetailRouteView"),
				},
			},
			{
				Key: "benchmark.UpdateRoute",
				Value: lago.Route{
					Path:    "PUT /api/articles/{id}/",
					Handler: lago.NewDynamicView("benchmark.UpdateRouteView"),
				},
			},
			{
				Key: "benchmark.DeleteRoute",
				Value: lago.Route{
					Path:    "DELETE /api/articles/{id}/",
					Handler: lago.NewDynamicView("benchmark.DeleteRouteView"),
				},
			},
			{
				Key: "benchmark.TruncateRoute",
				Value: lago.Route{
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
				Value: lago.Route{
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
				Value: lago.Route{
					Path:    "GET /api/ws/",
					Handler: http.HandlerFunc(BenchmarkWSHandler),
				},
			},
		},
	}
}

