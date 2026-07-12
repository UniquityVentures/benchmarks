package benchmark

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/UniquityVentures/lamu/components"
	"github.com/UniquityVentures/lamu/getters"
	"github.com/UniquityVentures/lamu/lamu"
	"github.com/UniquityVentures/lamu/registry"
	"github.com/UniquityVentures/lamu/views"
	"maragu.dev/gomponents"
)

// DummyForm implements components.FormInterface to parse JSON requests
type DummyForm struct {
	components.Page
}

func (DummyForm) GetKey() string { return "dummy_form" }

func (DummyForm) GetRoles() []string { return nil }

func (DummyForm) Build(ctx context.Context) gomponents.Node {
	return nil
}

func (DummyForm) ParseForm(r *http.Request) (map[string]any, map[string]error, error) {
	if r.Method == http.MethodGet || r.ContentLength <= 0 {
		return map[string]any{}, nil, nil
	}
	var input map[string]any
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		return nil, nil, err
	}
	return input, nil, nil
}

// DummyPage implements components.ParentInterface to return DummyForm as child
type DummyPage struct {
	components.Page
}

func (DummyPage) GetKey() string { return "dummy_page" }

func (DummyPage) GetRoles() []string { return nil }

func (DummyPage) Build(ctx context.Context) gomponents.Node {
	return nil
}

func (DummyPage) GetChildren() []components.PageInterface {
	return []components.PageInterface{DummyForm{}}
}

// HTTPMethodOverrideLayer translates PUT/DELETE to POST for LayerUpdate and LayerDelete
type HTTPMethodOverrideLayer struct{}

func (HTTPMethodOverrideLayer) Next(view views.View, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut || r.Method == http.MethodDelete {
			r.Method = http.MethodPost
		}
		next.ServeHTTP(w, r)
	})
}

// JSONResponseLayer is a generic layer to handle final JSON response serializations
type JSONResponseLayer[T any] struct {
	Key    string
	Status int
}

func (l JSONResponseLayer[T]) Next(view views.View, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		w.Header().Set("Content-Type", "application/json")
		if l.Status != 0 {
			w.WriteHeader(l.Status)
		}
		if l.Status == http.StatusNoContent {
			return
		}

		val := ctx.Value(l.Key)

		// 1. Handle slice response from ObjectList
		if objectList, ok := val.(components.ObjectList[T]); ok {
			if objectList.Items == nil {
				json.NewEncoder(w).Encode([]T{})
			} else {
				json.NewEncoder(w).Encode(objectList.Items)
			}
			return
		}

		// 2. Handle DB loading for newly created records
		if l.Key == "$id" {
			id, _ := val.(uint)
			db, _ := getters.DBFromContext(ctx)
			var record T
			db.First(&record, id)
			json.NewEncoder(w).Encode(record)
			return
		}

		// 3. Handle update re-fetching
		if l.Key == "article" {
			article, _ := val.(Article)
			db, _ := getters.DBFromContext(ctx)
			var updatedArticle Article
			db.First(&updatedArticle, article.ID)
			json.NewEncoder(w).Encode(updatedArticle)
			return
		}

		json.NewEncoder(w).Encode(val)
	})
}

func pluginViews() lamu.PluginFeatures[*views.View] {
	return lamu.PluginFeatures[*views.View]{
		Entries: []registry.Pair[string, *views.View]{
			{
				Key: "benchmark.ListRouteView",
				Value: &views.View{
					PageName: "dummy_page",
					PageLookup: func(name string) (components.PageInterface, bool) {
						return DummyPage{}, true
					},
					Layers: []registry.Pair[string, views.Layer]{
						registry.Pair[string, views.Layer]{
							Key: "list",
							Value: views.LayerList[Article]{
								Key: getters.Static("articles"),
							},
						},
						registry.Pair[string, views.Layer]{
							Key: "json_response",
							Value: JSONResponseLayer[Article]{
								Key: "articles",
							},
						},
					},
				},
			},
			{
				Key: "benchmark.CreateRouteView",
				Value: &views.View{
					PageName: "dummy_page",
					PageLookup: func(name string) (components.PageInterface, bool) {
						return DummyPage{}, true
					},
					Layers: []registry.Pair[string, views.Layer]{
						registry.Pair[string, views.Layer]{
							Key:   "create",
							Value: views.LayerCreate[Article]{},
						},
						registry.Pair[string, views.Layer]{
							Key: "json_response",
							Value: JSONResponseLayer[Article]{
								Key:    "$id",
								Status: http.StatusCreated,
							},
						},
					},
				},
			},
			{
				Key: "benchmark.DetailRouteView",
				Value: &views.View{
					PageName: "dummy_page",
					PageLookup: func(name string) (components.PageInterface, bool) {
						return DummyPage{}, true
					},
					Layers: []registry.Pair[string, views.Layer]{
						registry.Pair[string, views.Layer]{
							Key: "detail",
							Value: views.LayerDetail[Article]{
								Key:          getters.Static("article"),
								PathParamKey: getters.Static("id"),
							},
						},
						registry.Pair[string, views.Layer]{
							Key: "json_response",
							Value: JSONResponseLayer[Article]{
								Key: "article",
							},
						},
					},
				},
			},
			{
				Key: "benchmark.UpdateRouteView",
				Value: &views.View{
					PageName: "dummy_page",
					PageLookup: func(name string) (components.PageInterface, bool) {
						return DummyPage{}, true
					},
					Layers: []registry.Pair[string, views.Layer]{
						registry.Pair[string, views.Layer]{
							Key:   "override",
							Value: HTTPMethodOverrideLayer{},
						},
						registry.Pair[string, views.Layer]{
							Key: "detail",
							Value: views.LayerDetail[Article]{
								Key:          getters.Static("article"),
								PathParamKey: getters.Static("id"),
							},
						},
						registry.Pair[string, views.Layer]{
							Key: "update",
							Value: views.LayerUpdate[Article]{
								Key: getters.Static("article"),
							},
						},
						registry.Pair[string, views.Layer]{
							Key: "json_response",
							Value: JSONResponseLayer[Article]{
								Key: "article",
							},
						},
					},
				},
			},
			{
				Key: "benchmark.DeleteRouteView",
				Value: &views.View{
					PageName: "dummy_page",
					PageLookup: func(name string) (components.PageInterface, bool) {
						return DummyPage{}, true
					},
					Layers: []registry.Pair[string, views.Layer]{
						registry.Pair[string, views.Layer]{
							Key:   "override",
							Value: HTTPMethodOverrideLayer{},
						},
						registry.Pair[string, views.Layer]{
							Key: "detail",
							Value: views.LayerDetail[Article]{
								Key:          getters.Static("article"),
								PathParamKey: getters.Static("id"),
							},
						},
						registry.Pair[string, views.Layer]{
							Key: "delete",
							Value: views.LayerDelete[Article]{
								Key: getters.Static("article"),
							},
						},
						registry.Pair[string, views.Layer]{
							Key: "json_response",
							Value: JSONResponseLayer[Article]{
								Status: http.StatusNoContent,
							},
						},
					},
				},
			},
		},
	}
}
