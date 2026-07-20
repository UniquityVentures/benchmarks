package benchmark

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type WSSmallResponse struct {
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
	ClientID  string `json:"client_id"`
	Seq       int    `json:"seq"`
	Payload   string `json:"payload"`
}

type WSMetric struct {
	ID     int     `json:"id"`
	Name   string  `json:"name"`
	Value  float64 `json:"value"`
	Status string  `json:"status"`
}

type WSMediumResponse struct {
	Type      string            `json:"type"`
	Timestamp int64             `json:"timestamp"`
	Meta      map[string]string `json:"meta"`
	Metrics   []WSMetric        `json:"metrics"`
}

type WSAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type WSRecord struct {
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Tags   []string `json:"tags"`
	Author WSAuthor `json:"author"`
}

type WSLargeResponse struct {
	Type         string     `json:"type"`
	SyncID       string     `json:"sync_id"`
	RecordsCount int        `json:"records_count"`
	Records      []WSRecord `json:"records"`
}

type WSRequest struct {
	Query string          `json:"query"`
	Data  json.RawMessage `json:"data"`
}

var (
	smallResponse  WSSmallResponse
	mediumResponse WSMediumResponse
	largeResponse  WSLargeResponse
)

func init() {
	smallResponse = WSSmallResponse{
		Type:      "ping",
		Timestamp: 1783993200123,
		ClientID:  "client_8b31a",
		Seq:       1024,
		Payload:   "hello",
	}

	meta := map[string]string{
		"session_id": "sess_812da1823abf",
		"user_role":  "editor",
		"version":    "1.4.0",
	}
	metrics := make([]WSMetric, 50)
	for i := 0; i < 50; i++ {
		metrics[i] = WSMetric{
			ID:     i + 1,
			Name:   fmt.Sprintf("metric_name_indicator_%d", i),
			Value:  float64(i) * 1.5,
			Status: "ok",
		}
	}
	mediumResponse = WSMediumResponse{
		Type:      "dashboard_update",
		Timestamp: 1783993200123,
		Meta:      meta,
		Metrics:   metrics,
	}

	records := make([]WSRecord, 500)
	dummyBody := "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."
	for i := 0; i < 500; i++ {
		records[i] = WSRecord{
			ID:     fmt.Sprintf("rec_%04d", i+1),
			Title:  fmt.Sprintf("Random Article Title %04d", i+1),
			Body:   dummyBody,
			Tags:   []string{"performance", "benchmark", "websocket", "go"},
			Author: WSAuthor{
				Name:  "Jane Doe",
				Email: "jane.doe@example.com",
			},
		}
	}
	largeResponse = WSLargeResponse{
		Type:         "bulk_sync",
		SyncID:       "sync_91238ba18",
		RecordsCount: 500,
		Records:      records,
	}
}

func BenchmarkWSHandler(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}
	defer c.Close(websocket.StatusInternalError, "closing")

	// Disable read limit to allow handling large request payloads (> 32KB)
	c.SetReadLimit(-1)

	ctx := r.Context()
	for {
		var req WSRequest
		if err := wsjson.Read(ctx, c, &req); err != nil {
			break
		}
		var err error
		if req.Query == "medium" {
			err = wsjson.Write(ctx, c, mediumResponse)
		} else if req.Query == "large" {
			err = wsjson.Write(ctx, c, largeResponse)
		} else {
			err = wsjson.Write(ctx, c, smallResponse)
		}
		if err != nil {
			break
		}
	}
}

