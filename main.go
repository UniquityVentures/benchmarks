package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"image/color"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/websocket"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

type WSSmallPayload struct {
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

type WSMediumPayload struct {
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

type WSLargePayload struct {
	Type         string     `json:"type"`
	SyncID       string     `json:"sync_id"`
	RecordsCount int        `json:"records_count"`
	Records      []WSRecord `json:"records"`
}

type WSRequest struct {
	Query string `json:"query"`
	Data  any    `json:"data"`
}

var (
	clientSmallPayload  WSSmallPayload
	clientMediumPayload WSMediumPayload
	clientLargePayload  WSLargePayload
)

func init() {
	clientSmallPayload = WSSmallPayload{
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
	clientMediumPayload = WSMediumPayload{
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
	clientLargePayload = WSLargePayload{
		Type:         "bulk_sync",
		SyncID:       "sync_91238ba18",
		RecordsCount: 500,
		Records:      records,
	}
}

// TargetConfig represents a benchmark target with a Name and URL.
type TargetConfig struct {
	Name string `toml:"name"`
	URL  string `toml:"url"`
}

// TomlConfig represents the structure of the config.toml file.
type TomlConfig struct {
	Targets   []TargetConfig `toml:"targets"`
	Durations []string       `toml:"durations"`
	Cooldowns []string       `toml:"cooldowns"`
	Workers   []int          `toml:"workers"`
}

// Config represents a benchmarking configuration permutation.
type Config struct {
	Target     TargetConfig
	Duration   time.Duration
	Cooldown   time.Duration
	NumWorkers int
}

// SubConfig represents the non-URL components of a Config.
type SubConfig struct {
	Duration   time.Duration
	Cooldown   time.Duration
	NumWorkers int
}

// BenchmarkStats represents the collected metrics for a benchmark run.
type BenchmarkStats struct {
	MaxConnections     int64
	AvgConnections     float64
	MaxRPS             float64
	AvgRPS             float64
	MaxLatency         time.Duration
	AvgLatency         time.Duration
	AvgBytesSent       float64
	MaxBytesSent       int64
	TotalBytesSent     int64
	AvgBytesReceived   float64
	MaxBytesReceived   int64
	TotalBytesReceived int64
	TotalRequests      int64
	SuccessRequests    int64
	FailedRequests     int64
	Errors             int64
}

// SafeIDPool is a thread-safe pool of article IDs.
type SafeIDPool struct {
	mu  sync.RWMutex
	ids []uint
}

func NewSafeIDPool() *SafeIDPool {
	return &SafeIDPool{
		ids: make([]uint, 0),
	}
}

func (p *SafeIDPool) Add(id uint) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ids = append(p.ids, id)
}

func (p *SafeIDPool) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.ids)
}

func (p *SafeIDPool) GetRandomNewest(limit int) (uint, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	n := len(p.ids)
	if n == 0 {
		return 0, false
	}
	start := n - limit
	if start < 0 {
		start = 0
	}
	// pick a random index between start and n-1
	idx := start + rand.Intn(n-start)
	return p.ids[idx], true
}

func (p *SafeIDPool) RemoveOldest() (uint, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.ids) == 0 {
		return 0, false
	}
	val := p.ids[0]
	p.ids = p.ids[1:]
	return val, true
}

// watchedConn wraps net.Conn to track open/closed TCP connections.
type watchedConn struct {
	net.Conn
	activeCount *int32
	closed      int32
}

func (w *watchedConn) Close() error {
	if atomic.CompareAndSwapInt32(&w.closed, 0, 1) {
		atomic.AddInt32(w.activeCount, -1)
	}
	return w.Conn.Close()
}

// WorkerMetrics keeps track of metrics inside worker goroutines.
type WorkerMetrics struct {
	TotalRequests      int64
	SuccessRequests    int64
	FailedRequests     int64
	Errors             int64
	TotalBytesSent     int64
	TotalBytesReceived int64
	MaxBytesSent       int64
	MaxBytesReceived   int64
	MaxLatency         int64 // in nanoseconds
	TotalLatency       int64 // in nanoseconds
}

func loadConfig(path string) ([]TargetConfig, []time.Duration, []time.Duration, []int, error) {
	var tomlConf TomlConfig
	if _, err := toml.DecodeFile(path, &tomlConf); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to decode TOML file: %w", err)
	}

	if len(tomlConf.Targets) == 0 {
		return nil, nil, nil, nil, fmt.Errorf("no targets specified in configuration")
	}

	var parsedDurations []time.Duration
	for _, s := range tomlConf.Durations {
		d, err := time.ParseDuration(s)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("invalid duration %q: %w", s, err)
		}
		parsedDurations = append(parsedDurations, d)
	}
	if len(parsedDurations) == 0 {
		parsedDurations = []time.Duration{5 * time.Second}
	}

	var parsedCooldowns []time.Duration
	for _, s := range tomlConf.Cooldowns {
		d, err := time.ParseDuration(s)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("invalid cooldown %q: %w", s, err)
		}
		parsedCooldowns = append(parsedCooldowns, d)
	}
	if len(parsedCooldowns) == 0 {
		parsedCooldowns = []time.Duration{2 * time.Second}
	}

	var parsedWorkers []int
	for _, w := range tomlConf.Workers {
		if w <= 0 {
			return nil, nil, nil, nil, fmt.Errorf("invalid worker count %d: must be positive", w)
		}
		parsedWorkers = append(parsedWorkers, w)
	}
	if len(parsedWorkers) == 0 {
		parsedWorkers = []int{10}
	}

	return tomlConf.Targets, parsedDurations, parsedCooldowns, parsedWorkers, nil
}

func main() {
	configPath := flag.String("config", "config.toml", "Path to the TOML configuration file")
	flag.Parse()

	log.Println("Starting Go Benchmarking Utility")
	log.Printf("Loading configuration from: %s", *configPath)

	targets, durations, cooldowns, workers, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	log.Printf("Loaded Targets:   %d", len(targets))
	for _, t := range targets {
		log.Printf(" - %s: %s", t.Name, t.URL)
	}
	log.Printf("Durations:  %v", durations)
	log.Printf("Cooldowns:  %v", cooldowns)
	log.Printf("Workers:    %v", workers)

	results := make(map[Config]BenchmarkStats)
	var subConfigs []SubConfig

	// Generate sub-configs list for tracking and plotting order
	for _, duration := range durations {
		for _, cooldown := range cooldowns {
			for _, numWorkers := range workers {
				subConfigs = append(subConfigs, SubConfig{
					Duration:   duration,
					Cooldown:   cooldown,
					NumWorkers: numWorkers,
				})
			}
		}
	}

	// Permutation loop for CRUD
	for _, target := range targets {
		for _, sub := range subConfigs {
			config := Config{
				Target:     target,
				Duration:   sub.Duration,
				Cooldown:   sub.Cooldown,
				NumWorkers: sub.NumWorkers,
			}

			log.Printf("================================================================================")
			log.Printf("STARTING CRUD BENCHMARK: Target=%s (%s) | Workers=%d | Duration=%s | Cooldown=%s",
				config.Target.Name, config.Target.URL, config.NumWorkers, config.Duration, config.Cooldown)

			stats := runBenchmark(config)
			results[config] = stats

			log.Printf("CRUD BENCHMARK FINISHED. Requests: %d | Avg RPS: %.2f | Avg Latency: %v | Avg Conn: %.2f",
				stats.TotalRequests, stats.AvgRPS, stats.AvgLatency, stats.AvgConnections)

			// Cooldown logic with DB truncation
			log.Printf("Sleeping for 1 second before database truncation...")
			time.Sleep(1 * time.Second)

			truncateURL := config.Target.URL + "/api/truncate/"
			log.Printf("Truncating database via endpoint: %s", truncateURL)
			if err := sendTruncateRequest(truncateURL); err != nil {
				log.Printf("Warning: failed to truncate database: %v", err)
			} else {
				log.Println("Database truncated successfully.")
			}

			remainingCooldown := config.Cooldown - 1*time.Second
			if remainingCooldown > 0 {
				log.Printf("Sleeping for remaining cooldown: %s", remainingCooldown)
				time.Sleep(remainingCooldown)
			}
		}
	}

	// Permutation loop for Counter
	counterResults := make(map[Config]BenchmarkStats)
	for _, target := range targets {
		for _, sub := range subConfigs {
			config := Config{
				Target:     target,
				Duration:   sub.Duration,
				Cooldown:   sub.Cooldown,
				NumWorkers: sub.NumWorkers,
			}

			log.Printf("================================================================================")
			log.Printf("STARTING COUNTER BENCHMARK: Target=%s (%s) | Workers=%d | Duration=%s | Cooldown=%s",
				config.Target.Name, config.Target.URL, config.NumWorkers, config.Duration, config.Cooldown)

			stats := runCounterBenchmark(config)
			counterResults[config] = stats

			log.Printf("COUNTER BENCHMARK FINISHED. Requests: %d | Avg RPS: %.2f | Avg Latency: %v | Avg Conn: %.2f",
				stats.TotalRequests, stats.AvgRPS, stats.AvgLatency, stats.AvgConnections)

			if config.Cooldown > 0 {
				log.Printf("Sleeping for cooldown: %s", config.Cooldown)
				time.Sleep(config.Cooldown)
			}
		}
	}

	// Permutation loop for WebSockets (9 stages)
	wsResults := make(map[string]map[Config]BenchmarkStats)
	wsStages := []struct {
		client string
		server string
	}{
		{"small", "small"},
		{"small", "medium"},
		{"small", "large"},
		{"medium", "small"},
		{"medium", "medium"},
		{"medium", "large"},
		{"large", "small"},
		{"large", "medium"},
		{"large", "large"},
	}

	for _, stage := range wsStages {
		stageName := fmt.Sprintf("WS_%s_req_%s_resp", stage.client, stage.server)
		wsResults[stageName] = make(map[Config]BenchmarkStats)
		for _, target := range targets {
			// Skip WSGI targets as WSGI fundamentally does not support WebSockets
			if strings.Contains(strings.ToLower(target.Name), "wsgi") {
				continue
			}
			for _, sub := range subConfigs {
				config := Config{
					Target:     target,
					Duration:   sub.Duration,
					Cooldown:   sub.Cooldown,
					NumWorkers: sub.NumWorkers,
				}

				log.Printf("================================================================================")
				log.Printf("STARTING WEBSOCKET BENCHMARK (%s): Target=%s (%s) | Workers=%d | Duration=%s | Cooldown=%s",
					stageName, config.Target.Name, config.Target.URL, config.NumWorkers, config.Duration, config.Cooldown)

				stats := runWebsocketBenchmark(config, stage.client, stage.server)
				wsResults[stageName][config] = stats

				log.Printf("WEBSOCKET BENCHMARK (%s) FINISHED. Requests: %d | Avg RPS: %.2f | Avg Latency: %v | Avg Conn: %.2f",
					stageName, stats.TotalRequests, stats.AvgRPS, stats.AvgLatency, stats.AvgConnections)

				if config.Cooldown > 0 {
					log.Printf("Sleeping for cooldown: %s", config.Cooldown)
					time.Sleep(config.Cooldown)
				}
			}
		}
	}

	// Plot results
	log.Println("================================================================================")
	log.Println("Generating SVG Plots for CRUD...")
	metricsToPlot := []struct {
		name   string
		yLabel string
		file   string
	}{
		{"Average RPS", "Requests Per Second", "average_rps.svg"},
		{"Max RPS", "Requests Per Second", "max_rps.svg"},
		{"Average Latency (ms)", "Latency (ms)", "average_latency.svg"},
		{"Max Latency (ms)", "Latency (ms)", "max_latency.svg"},
		{"Average Connections", "TCP Connections", "average_connections.svg"},
		{"Max Connections", "TCP Connections", "max_connections.svg"},
		{"Total Bytes Received (MB)", "Data Received (MB)", "total_bytes_received.svg"},
		{"Average Bytes Received (KB)", "Data Received (KB)", "average_bytes_received.svg"},
	}

	for _, m := range metricsToPlot {
		if err := plotMetric(m.name, m.yLabel, m.file, targets, subConfigs, results); err != nil {
			log.Fatalf("Error plotting %q: %v", m.name, err)
		}
		log.Printf("Saved plot: %s", m.file)
	}

	log.Println("Generating SVG Plots for Counter...")
	counterMetricsToPlot := []struct {
		name   string
		yLabel string
		file   string
	}{
		{"Average RPS", "Requests Per Second", "counter_average_rps.svg"},
		{"Max RPS", "Requests Per Second", "counter_max_rps.svg"},
		{"Average Latency (ms)", "Latency (ms)", "counter_average_latency.svg"},
		{"Max Latency (ms)", "Latency (ms)", "counter_max_latency.svg"},
		{"Average Connections", "TCP Connections", "counter_average_connections.svg"},
		{"Max Connections", "TCP Connections", "counter_max_connections.svg"},
		{"Total Bytes Received (MB)", "Data Received (MB)", "counter_total_bytes_received.svg"},
		{"Average Bytes Received (KB)", "Data Received (KB)", "counter_average_bytes_received.svg"},
	}

	for _, m := range counterMetricsToPlot {
		if err := plotMetric(m.name, m.yLabel, m.file, targets, subConfigs, counterResults); err != nil {
			log.Fatalf("Error plotting %q: %v", m.name, err)
		}
		log.Printf("Saved plot: %s", m.file)
	}

	// Generate WS plots for each worker count configuration
	log.Println("Generating SVG Plots for WebSockets...")
	for _, subConfig := range subConfigs {
		// Only plot one duration configuration to avoid duplicate charts
		if len(durations) > 0 && subConfig.Duration != durations[0] {
			continue
		}

		rpsFile := fmt.Sprintf("websocket_average_rps_w%d.svg", subConfig.NumWorkers)
		if err := plotWSMetric("Average RPS", "Requests Per Second", rpsFile, targets, subConfig, wsStages, wsResults); err != nil {
			log.Fatalf("Error plotting WS RPS: %v", err)
		}
		log.Printf("Saved plot: %s", rpsFile)

		latFile := fmt.Sprintf("websocket_average_latency_w%d.svg", subConfig.NumWorkers)
		if err := plotWSMetric("Average Latency (ms)", "Latency (ms)", latFile, targets, subConfig, wsStages, wsResults); err != nil {
			log.Fatalf("Error plotting WS Latency: %v", err)
		}
		log.Printf("Saved plot: %s", latFile)
	}

	log.Println("All benchmarks completed successfully and plots generated.")
	saveMetricsJSON("benchmark_metrics.json", results, counterResults, wsResults)
}

func runBenchmark(config Config) BenchmarkStats {
	var activeTCPConns int32
	var maxTCPConns atomic.Int32

	// Setup custom fasthttp client to monitor connections
	client := &fasthttp.Client{
		Name: "go-benchmark-client",
		Dial: func(addr string) (net.Conn, error) {
			conn, err := fasthttp.Dial(addr)
			if err != nil {
				return nil, err
			}
			atomic.AddInt32(&activeTCPConns, 1)

			// Update maxTCPConns
			for {
				curMax := maxTCPConns.Load()
				curActive := atomic.LoadInt32(&activeTCPConns)
				if curActive <= curMax {
					break
				}
				if maxTCPConns.CompareAndSwap(curMax, curActive) {
					break
				}
			}

			return &watchedConn{
				Conn:        conn,
				activeCount: &activeTCPConns,
			}, nil
		},
	}

	idPool := NewSafeIDPool()
	metrics := &WorkerMetrics{}

	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)
	defer cancel()

	// Track RPS buckets (1-second intervals)
	numSeconds := max(int(config.Duration.Seconds()), 1)
	rpsBuckets := make([]int64, numSeconds)
	benchStart := time.Now()

	// Start connection sampler
	var connSamplesSum int64
	var connSamplesCount int64
	var connSamplesMu sync.Mutex
	sampleCtx, sampleCancel := context.WithCancel(context.Background())

	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-sampleCtx.Done():
				return
			case <-ticker.C:
				val := atomic.LoadInt32(&activeTCPConns)
				connSamplesMu.Lock()
				connSamplesSum += int64(val)
				connSamplesCount++
				connSamplesMu.Unlock()
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Add(config.NumWorkers)

	for i := 0; i < config.NumWorkers; i++ {
		workerRand := rand.New(rand.NewSource(time.Now().UnixNano() + int64(i)))
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					executeRequest(client, config.Target.URL, idPool, workerRand, metrics, benchStart, rpsBuckets)
				}
			}
		}()
	}

	wg.Wait()
	sampleCancel()

	// Compute finalized stats
	totalReqs := atomic.LoadInt64(&metrics.TotalRequests)
	avgRPS := float64(totalReqs) / config.Duration.Seconds()

	maxRPS := float64(0)
	for _, val := range rpsBuckets {
		if float64(val) > maxRPS {
			maxRPS = float64(val)
		}
	}

	avgLatency := time.Duration(0)
	if totalReqs > 0 {
		avgLatency = time.Duration(atomic.LoadInt64(&metrics.TotalLatency) / totalReqs)
	}

	avgConns := float64(0)
	connSamplesMu.Lock()
	if connSamplesCount > 0 {
		avgConns = float64(connSamplesSum) / float64(connSamplesCount)
	}
	connSamplesMu.Unlock()

	avgBytesSent := float64(0)
	if totalReqs > 0 {
		avgBytesSent = float64(atomic.LoadInt64(&metrics.TotalBytesSent)) / float64(totalReqs)
	}

	avgBytesRecv := float64(0)
	if totalReqs > 0 {
		avgBytesRecv = float64(atomic.LoadInt64(&metrics.TotalBytesReceived)) / float64(totalReqs)
	}

	return BenchmarkStats{
		MaxConnections:     int64(maxTCPConns.Load()),
		AvgConnections:     avgConns,
		MaxRPS:             maxRPS,
		AvgRPS:             avgRPS,
		MaxLatency:         time.Duration(atomic.LoadInt64(&metrics.MaxLatency)),
		AvgLatency:         avgLatency,
		AvgBytesSent:       avgBytesSent,
		MaxBytesSent:       atomic.LoadInt64(&metrics.MaxBytesSent),
		TotalBytesSent:     atomic.LoadInt64(&metrics.TotalBytesSent),
		AvgBytesReceived:   avgBytesRecv,
		MaxBytesReceived:   atomic.LoadInt64(&metrics.MaxBytesReceived),
		TotalBytesReceived: atomic.LoadInt64(&metrics.TotalBytesReceived),
		TotalRequests:      totalReqs,
		SuccessRequests:    atomic.LoadInt64(&metrics.SuccessRequests),
		FailedRequests:     atomic.LoadInt64(&metrics.FailedRequests),
		Errors:             atomic.LoadInt64(&metrics.Errors),
	}
}

func updateMax(maxVal *int64, val int64) {
	for {
		old := atomic.LoadInt64(maxVal)
		if val <= old {
			break
		}
		if atomic.CompareAndSwapInt64(maxVal, old, val) {
			break
		}
	}
}

func randomString(r *rand.Rand, length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = chars[r.Intn(len(chars))]
	}
	return string(b)
}

func doRequest(
	client *fasthttp.Client,
	method, urlStr string,
	body []byte,
	isJSON bool,
	metrics *WorkerMetrics,
	benchStart time.Time,
	rpsBuckets []int64,
) (int, []byte) {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.Header.SetMethod(method)
	req.SetRequestURI(urlStr)
	req.Header.Set("Cookie", "environment=%7B%7D")
	req.Header.Set("Accept-Encoding", "gzip")
	if isJSON {
		req.Header.SetContentType("application/json")
		if len(body) > 0 {
			req.SetBody(body)
		}
	}

	start := time.Now()
	err := client.Do(req, resp)
	latency := time.Since(start)

	if err != nil {
		atomic.AddInt64(&metrics.Errors, 1)
		return 0, nil
	}

	// Calculate HTTP bytes sent (Request header + Request body)
	headerBytesSent, _ := req.Header.WriteTo(io.Discard)
	bytesSent := headerBytesSent + int64(len(req.Body()))

	// Calculate HTTP bytes received (Response header + Response body)
	headerBytesReceived, _ := resp.Header.WriteTo(io.Discard)
	bytesReceived := headerBytesReceived + int64(len(resp.Body()))

	statusCode := resp.StatusCode()
	var respBody []byte
	contentEncoding := resp.Header.Peek("Content-Encoding")
	if bytes.EqualFold(contentEncoding, []byte("gzip")) {
		var err error
		respBody, err = resp.BodyGunzip()
		if err != nil {
			atomic.AddInt64(&metrics.Errors, 1)
			return 0, nil
		}
	} else {
		respBody = resp.Body()
	}

	// Update atomic metrics
	atomic.AddInt64(&metrics.TotalRequests, 1)
	atomic.AddInt64(&metrics.TotalBytesSent, bytesSent)
	atomic.AddInt64(&metrics.TotalBytesReceived, bytesReceived)
	updateMax(&metrics.MaxBytesSent, bytesSent)
	updateMax(&metrics.MaxBytesReceived, bytesReceived)
	updateMax(&metrics.MaxLatency, int64(latency))
	atomic.AddInt64(&metrics.TotalLatency, int64(latency))

	if statusCode >= 200 && statusCode < 300 {
		atomic.AddInt64(&metrics.SuccessRequests, 1)
	} else {
		atomic.AddInt64(&metrics.FailedRequests, 1)
	}

	// Keep track of RPS bucket
	bucket := int(time.Since(benchStart).Seconds())
	if bucket >= 0 && bucket < len(rpsBuckets) {
		atomic.AddInt64(&rpsBuckets[bucket], 1)
	}

	var bodyCopy []byte
	if statusCode == fasthttp.StatusCreated {
		bodyCopy = make([]byte, len(respBody))
		copy(bodyCopy, respBody)
	}

	return statusCode, bodyCopy
}

func executeRequest(
	client *fasthttp.Client,
	baseURL string,
	pool *SafeIDPool,
	r *rand.Rand,
	metrics *WorkerMetrics,
	benchStart time.Time,
	rpsBuckets []int64,
) {
	workflow := r.Intn(3) + 1

	switch workflow {
	case 1:
		action := r.Intn(3) + 1
		if action == 1 || pool.Len() == 0 {
			// Create
			title := fmt.Sprintf("W1_%s", randomString(r, 5))
			content := fmt.Sprintf("Content_%s", randomString(r, 10))
			body := fmt.Sprintf(`{"title":"%s","content":"%s"}`, title, content)
			status, respBody := doRequest(client, "POST", baseURL+"/api/articles/", []byte(body), true, metrics, benchStart, rpsBuckets)
			if status == fasthttp.StatusCreated && respBody != nil {
				var resp struct {
					ID uint `json:"id"`
				}
				if err := json.Unmarshal(respBody, &resp); err == nil && resp.ID > 0 {
					pool.Add(resp.ID)
				}
			}
		} else if action == 2 {
			// Update random
			id, ok := pool.GetRandomNewest(50)
			if ok && id > 0 {
				title := fmt.Sprintf("W1_UPD_%s", randomString(r, 5))
				content := fmt.Sprintf("Content_UPD_%s", randomString(r, 10))
				body := fmt.Sprintf(`{"title":"%s","content":"%s"}`, title, content)
				url := fmt.Sprintf("%s/api/articles/%d/", baseURL, id)
				doRequest(client, "PUT", url, []byte(body), true, metrics, benchStart, rpsBuckets)
			} else {
				// Fallback to Create
				title := fmt.Sprintf("W1_%s", randomString(r, 5))
				content := fmt.Sprintf("Content_%s", randomString(r, 10))
				body := fmt.Sprintf(`{"title":"%s","content":"%s"}`, title, content)
				status, respBody := doRequest(client, "POST", baseURL+"/api/articles/", []byte(body), true, metrics, benchStart, rpsBuckets)
				if status == fasthttp.StatusCreated && respBody != nil {
					var resp struct {
						ID uint `json:"id"`
					}
					if err := json.Unmarshal(respBody, &resp); err == nil && resp.ID > 0 {
						pool.Add(resp.ID)
					}
				}
			}
		} else {
			// Delete oldest
			if pool.Len() > 100 {
				id, ok := pool.RemoveOldest()
				if ok && id > 0 {
					url := fmt.Sprintf("%s/api/articles/%d/", baseURL, id)
					doRequest(client, "DELETE", url, nil, false, metrics, benchStart, rpsBuckets)
				}
			} else {
				// Fallback to Create
				title := fmt.Sprintf("W1_%s", randomString(r, 5))
				content := fmt.Sprintf("Content_%s", randomString(r, 10))
				body := fmt.Sprintf(`{"title":"%s","content":"%s"}`, title, content)
				status, respBody := doRequest(client, "POST", baseURL+"/api/articles/", []byte(body), true, metrics, benchStart, rpsBuckets)
				if status == fasthttp.StatusCreated && respBody != nil {
					var resp struct {
						ID uint `json:"id"`
					}
					if err := json.Unmarshal(respBody, &resp); err == nil && resp.ID > 0 {
						pool.Add(resp.ID)
					}
				}
			}
		}
	case 2:
		action := r.Intn(3) + 1
		if action == 1 {
			// Create
			title := fmt.Sprintf("W2_%s", randomString(r, 5))
			content := fmt.Sprintf("Content_%s", randomString(r, 10))
			body := fmt.Sprintf(`{"title":"%s","content":"%s"}`, title, content)
			status, respBody := doRequest(client, "POST", baseURL+"/api/articles/", []byte(body), true, metrics, benchStart, rpsBuckets)
			if status == fasthttp.StatusCreated && respBody != nil {
				var resp struct {
					ID uint `json:"id"`
				}
				if err := json.Unmarshal(respBody, &resp); err == nil && resp.ID > 0 {
					pool.Add(resp.ID)
				}
			}
		} else if action == 2 {
			// List
			doRequest(client, "GET", baseURL+"/api/articles/", nil, false, metrics, benchStart, rpsBuckets)
		} else {
			// Delete oldest
			if pool.Len() > 100 {
				id, ok := pool.RemoveOldest()
				if ok && id > 0 {
					url := fmt.Sprintf("%s/api/articles/%d/", baseURL, id)
					doRequest(client, "DELETE", url, nil, false, metrics, benchStart, rpsBuckets)
				}
			} else {
				doRequest(client, "GET", baseURL+"/api/articles/", nil, false, metrics, benchStart, rpsBuckets)
			}
		}
	default:
		// Workflow 3
		action := r.Intn(3) + 1
		if action == 1 {
			// Create
			title := fmt.Sprintf("W3_%s", randomString(r, 5))
			content := fmt.Sprintf("Content_%s", randomString(r, 10))
			body := fmt.Sprintf(`{"title":"%s","content":"%s"}`, title, content)
			status, respBody := doRequest(client, "POST", baseURL+"/api/articles/", []byte(body), true, metrics, benchStart, rpsBuckets)
			if status == fasthttp.StatusCreated && respBody != nil {
				var resp struct {
					ID uint `json:"id"`
				}
				if err := json.Unmarshal(respBody, &resp); err == nil && resp.ID > 0 {
					pool.Add(resp.ID)
				}
			}
		} else if action == 2 {
			// List with filter
			filterChar := string(rune('a' + r.Intn(26)))
			doRequest(client, "GET", baseURL+"/api/articles/?title="+filterChar, nil, false, metrics, benchStart, rpsBuckets)
		} else {
			// Delete oldest
			if pool.Len() > 100 {
				id, ok := pool.RemoveOldest()
				if ok && id > 0 {
					url := fmt.Sprintf("%s/api/articles/%d/", baseURL, id)
					doRequest(client, "DELETE", url, nil, false, metrics, benchStart, rpsBuckets)
				}
			} else {
				doRequest(client, "GET", baseURL+"/api/articles/", nil, false, metrics, benchStart, rpsBuckets)
			}
		}
	}
}

func plotMetric(
	metricName string,
	yLabel string,
	filename string,
	targets []TargetConfig,
	subConfigs []SubConfig,
	results map[Config]BenchmarkStats,
) error {
	p := plot.New()
	p.Title.Text = metricName + " Comparison"
	p.Y.Label.Text = yLabel

	// Grid
	grid := plotter.NewGrid()
	grid.Vertical.Color = color.RGBA{R: 220, G: 220, B: 220, A: 255}
	grid.Horizontal.Color = color.RGBA{R: 220, G: 220, B: 220, A: 255}
	p.Add(grid)

	numSubConfigs := len(subConfigs)
	barWidth := vg.Points(12)

	// Modern elegant color palette
	premiumColors := []color.RGBA{
		{R: 79, G: 70, B: 229, A: 255},  // Indigo #4F46E5
		{R: 13, G: 148, B: 136, A: 255}, // Teal #0D9488
		{R: 219, G: 39, B: 119, A: 255}, // Pink/Rose #DB2777
		{R: 217, G: 119, B: 6, A: 255},  // Amber #D97706
		{R: 37, G: 99, B: 235, A: 255},  // Blue #2563EB
		{R: 124, G: 58, B: 237, A: 255}, // Violet #7C3AED
	}

	for i, subConfig := range subConfigs {
		values := make(plotter.Values, len(targets))
		for j, target := range targets {
			config := Config{
				Target:     target,
				Duration:   subConfig.Duration,
				Cooldown:   subConfig.Cooldown,
				NumWorkers: subConfig.NumWorkers,
			}
			stats := results[config]
			var val float64
			switch metricName {
			case "Average RPS":
				val = stats.AvgRPS
			case "Max RPS":
				val = stats.MaxRPS
			case "Average Latency (ms)":
				val = float64(stats.AvgLatency.Milliseconds())
			case "Max Latency (ms)":
				val = float64(stats.MaxLatency.Milliseconds())
			case "Average Connections":
				val = stats.AvgConnections
			case "Max Connections":
				val = float64(stats.MaxConnections)
			case "Total Bytes Received (MB)":
				val = float64(stats.TotalBytesReceived) / (1024.0 * 1024.0)
			case "Average Bytes Received (KB)":
				val = float64(stats.AvgBytesReceived) / 1024.0
			default:
				val = 0
			}
			values[j] = val
		}

		bars, err := plotter.NewBarChart(values, barWidth)
		if err != nil {
			return err
		}

		// Style
		colorIdx := i % len(premiumColors)
		bars.Color = premiumColors[colorIdx]
		bars.LineStyle.Color = premiumColors[colorIdx]
		bars.LineStyle.Width = vg.Points(1)

		// Offset
		offset := (float64(i) - float64(numSubConfigs-1)/2.0) * float64(barWidth)
		bars.Offset = vg.Points(offset)

		p.Add(bars)

		// Legend
		legendLabel := fmt.Sprintf("W:%d, D:%s", subConfig.NumWorkers, subConfig.Duration)
		p.Legend.Add(legendLabel, bars)
	}

	// Nominal X labels (using target names)
	targetNames := make([]string, len(targets))
	for i, t := range targets {
		targetNames[i] = t.Name
	}
	p.NominalX(targetNames...)

	plotWidth := vg.Points(float64(len(targets)*numSubConfigs*20 + 200))
	if plotWidth < 450 {
		plotWidth = 450
	}
	plotHeight := vg.Points(350)

	return p.Save(plotWidth, plotHeight, filename)
}

func sendTruncateRequest(urlStr string) error {
	client := &fasthttp.Client{
		Name: "benchmark-truncate-client",
	}
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.Header.SetMethod("POST")
	req.SetRequestURI(urlStr)

	err := client.Do(req, resp)
	if err != nil {
		return err
	}
	if resp.StatusCode() != fasthttp.StatusNoContent && resp.StatusCode() != fasthttp.StatusOK {
		return fmt.Errorf("unexpected status code: %d (body: %s)", resp.StatusCode(), resp.Body())
	}
	return nil
}

func executeCounterRequest(
	client *fasthttp.Client,
	baseURL string,
	metrics *WorkerMetrics,
	benchStart time.Time,
	rpsBuckets []int64,
) {
	body := []byte(`{"counter":42}`)
	doRequest(client, "POST", baseURL+"/api/counter/", body, true, metrics, benchStart, rpsBuckets)
}

func runCounterBenchmark(config Config) BenchmarkStats {
	var activeTCPConns int32
	var maxTCPConns atomic.Int32

	// Setup custom fasthttp client to monitor connections
	client := &fasthttp.Client{
		Name: "go-benchmark-client",
		Dial: func(addr string) (net.Conn, error) {
			conn, err := fasthttp.Dial(addr)
			if err != nil {
				return nil, err
			}
			atomic.AddInt32(&activeTCPConns, 1)

			// Update maxTCPConns
			for {
				curMax := maxTCPConns.Load()
				curActive := atomic.LoadInt32(&activeTCPConns)
				if curActive <= curMax {
					break
				}
				if maxTCPConns.CompareAndSwap(curMax, curActive) {
					break
				}
			}

			return &watchedConn{
				Conn:        conn,
				activeCount: &activeTCPConns,
			}, nil
		},
	}

	metrics := &WorkerMetrics{}

	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)
	defer cancel()

	// Track RPS buckets (1-second intervals)
	numSeconds := max(int(config.Duration.Seconds()), 1)
	rpsBuckets := make([]int64, numSeconds)
	benchStart := time.Now()

	// Start connection sampler
	var connSamplesSum int64
	var connSamplesCount int64
	var connSamplesMu sync.Mutex
	sampleCtx, sampleCancel := context.WithCancel(context.Background())

	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-sampleCtx.Done():
				return
			case <-ticker.C:
				val := atomic.LoadInt32(&activeTCPConns)
				connSamplesMu.Lock()
				connSamplesSum += int64(val)
				connSamplesCount++
				connSamplesMu.Unlock()
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Add(config.NumWorkers)

	for i := 0; i < config.NumWorkers; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					executeCounterRequest(client, config.Target.URL, metrics, benchStart, rpsBuckets)
				}
			}
		}()
	}

	wg.Wait()
	sampleCancel()

	// Compute finalized stats
	totalReqs := atomic.LoadInt64(&metrics.TotalRequests)
	avgRPS := float64(totalReqs) / config.Duration.Seconds()

	maxRPS := float64(0)
	for _, val := range rpsBuckets {
		if float64(val) > maxRPS {
			maxRPS = float64(val)
		}
	}

	avgLatency := time.Duration(0)
	if totalReqs > 0 {
		avgLatency = time.Duration(atomic.LoadInt64(&metrics.TotalLatency) / totalReqs)
	}

	avgConns := float64(0)
	connSamplesMu.Lock()
	if connSamplesCount > 0 {
		avgConns = float64(connSamplesSum) / float64(connSamplesCount)
	}
	connSamplesMu.Unlock()

	avgBytesSent := float64(0)
	if totalReqs > 0 {
		avgBytesSent = float64(atomic.LoadInt64(&metrics.TotalBytesSent)) / float64(totalReqs)
	}

	avgBytesRecv := float64(0)
	if totalReqs > 0 {
		avgBytesRecv = float64(atomic.LoadInt64(&metrics.TotalBytesReceived)) / float64(totalReqs)
	}

	return BenchmarkStats{
		MaxConnections:     int64(maxTCPConns.Load()),
		AvgConnections:     avgConns,
		MaxRPS:             maxRPS,
		AvgRPS:             avgRPS,
		MaxLatency:         time.Duration(atomic.LoadInt64(&metrics.MaxLatency)),
		AvgLatency:         avgLatency,
		AvgBytesSent:       avgBytesSent,
		MaxBytesSent:       atomic.LoadInt64(&metrics.MaxBytesSent),
		TotalBytesSent:     atomic.LoadInt64(&metrics.TotalBytesSent),
		AvgBytesReceived:   avgBytesRecv,
		MaxBytesReceived:   atomic.LoadInt64(&metrics.MaxBytesReceived),
		TotalBytesReceived: atomic.LoadInt64(&metrics.TotalBytesReceived),
		TotalRequests:      totalReqs,
		SuccessRequests:    atomic.LoadInt64(&metrics.SuccessRequests),
		FailedRequests:     atomic.LoadInt64(&metrics.FailedRequests),
		Errors:             atomic.LoadInt64(&metrics.Errors),
	}
}

func runWebsocketBenchmark(config Config, clientSize, serverSize string) BenchmarkStats {
	metrics := &WorkerMetrics{}

	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)
	defer cancel()

	// Track RPS buckets (1-second intervals)
	numSeconds := max(int(config.Duration.Seconds()), 1)
	rpsBuckets := make([]int64, numSeconds)
	benchStart := time.Now()

	// Determine client payload
	var dataPayload any
	switch clientSize {
	case "medium":
		dataPayload = clientMediumPayload
	case "large":
		dataPayload = clientLargePayload
	default:
		dataPayload = clientSmallPayload
	}

	reqPayload := WSRequest{
		Query: serverSize,
		Data:  dataPayload,
	}

	// Prepare raw payload bytes to avoid JSON marshal inside the loop
	reqBytes, err := json.Marshal(reqPayload)
	if err != nil {
		log.Fatalf("failed to marshal websocket request payload: %v", err)
	}

	var activeTCPConns int32
	var maxTCPConns atomic.Int32

	var wg sync.WaitGroup
	wg.Add(config.NumWorkers)

	for i := 0; i < config.NumWorkers; i++ {
		go func() {
			defer wg.Done()

			// Resolve WebSocket URL
			wsURL := config.Target.URL + "/api/ws/"
			if strings.HasPrefix(wsURL, "http://") {
				wsURL = "ws://" + strings.TrimPrefix(wsURL, "http://")
			} else if strings.HasPrefix(wsURL, "https://") {
				wsURL = "wss://" + strings.TrimPrefix(wsURL, "https://")
			}
			origin := config.Target.URL

			ws, err := websocket.Dial(wsURL, "", origin)
			if err != nil {
				errCount := atomic.AddInt64(&metrics.Errors, 1)
				if errCount == 1 {
					log.Printf("Websocket connection failed to %s: %v (first error shown)", wsURL, err)
				}
				return
			}
			defer ws.Close()

			atomic.AddInt32(&activeTCPConns, 1)
			for {
				curMax := maxTCPConns.Load()
				curActive := atomic.LoadInt32(&activeTCPConns)
				if curActive <= curMax {
					break
				}
				if maxTCPConns.CompareAndSwap(curMax, curActive) {
					break
				}
			}

			// Read/Write loop
			for {
				select {
				case <-ctx.Done():
					return
				default:
					start := time.Now()

					// Send raw bytes using Message codec
					err := websocket.Message.Send(ws, string(reqBytes))
					if err != nil {
						atomic.AddInt64(&metrics.Errors, 1)
						return
					}

					// Receive response
					var respStr string
					err = websocket.Message.Receive(ws, &respStr)
					if err != nil {
						atomic.AddInt64(&metrics.Errors, 1)
						return
					}

					latency := time.Since(start)

					// Calculate bytes sent/received
					bytesSent := int64(len(reqBytes))
					bytesReceived := int64(len(respStr))

					atomic.AddInt64(&metrics.TotalRequests, 1)
					atomic.AddInt64(&metrics.TotalBytesSent, bytesSent)
					atomic.AddInt64(&metrics.TotalBytesReceived, bytesReceived)
					updateMax(&metrics.MaxBytesSent, bytesSent)
					updateMax(&metrics.MaxBytesReceived, bytesReceived)
					updateMax(&metrics.MaxLatency, int64(latency))
					atomic.AddInt64(&metrics.TotalLatency, int64(latency))
					atomic.AddInt64(&metrics.SuccessRequests, 1)

					bucket := int(time.Since(benchStart).Seconds())
					if bucket >= 0 && bucket < len(rpsBuckets) {
						atomic.AddInt64(&rpsBuckets[bucket], 1)
					}
				}
			}
		}()
	}

	wg.Wait()

	// Compute finalized stats
	totalReqs := atomic.LoadInt64(&metrics.TotalRequests)
	avgRPS := float64(totalReqs) / config.Duration.Seconds()

	maxRPS := float64(0)
	for _, val := range rpsBuckets {
		if float64(val) > maxRPS {
			maxRPS = float64(val)
		}
	}

	avgLatency := time.Duration(0)
	if totalReqs > 0 {
		avgLatency = time.Duration(atomic.LoadInt64(&metrics.TotalLatency) / totalReqs)
	}

	avgBytesSent := float64(0)
	if totalReqs > 0 {
		avgBytesSent = float64(atomic.LoadInt64(&metrics.TotalBytesSent)) / float64(totalReqs)
	}

	avgBytesRecv := float64(0)
	if totalReqs > 0 {
		avgBytesRecv = float64(atomic.LoadInt64(&metrics.TotalBytesReceived)) / float64(totalReqs)
	}

	return BenchmarkStats{
		MaxConnections:     int64(maxTCPConns.Load()),
		AvgConnections:     float64(maxTCPConns.Load()),
		MaxRPS:             maxRPS,
		AvgRPS:             avgRPS,
		MaxLatency:         time.Duration(atomic.LoadInt64(&metrics.MaxLatency)),
		AvgLatency:         avgLatency,
		AvgBytesSent:       avgBytesSent,
		MaxBytesSent:       atomic.LoadInt64(&metrics.MaxBytesSent),
		TotalBytesSent:     atomic.LoadInt64(&metrics.TotalBytesSent),
		AvgBytesReceived:   avgBytesRecv,
		MaxBytesReceived:   atomic.LoadInt64(&metrics.MaxBytesReceived),
		TotalBytesReceived: atomic.LoadInt64(&metrics.TotalBytesReceived),
		TotalRequests:      totalReqs,
		SuccessRequests:    atomic.LoadInt64(&metrics.SuccessRequests),
		FailedRequests:     atomic.LoadInt64(&metrics.FailedRequests),
		Errors:             atomic.LoadInt64(&metrics.Errors),
	}
}

func plotWSMetric(
	metricName string,
	yLabel string,
	filename string,
	targets []TargetConfig,
	subConfig SubConfig,
	wsStages []struct{ client, server string },
	wsResults map[string]map[Config]BenchmarkStats,
) error {
	// Filter out WSGI targets since they don't support WebSockets
	var wsTargets []TargetConfig
	for _, target := range targets {
		if !strings.Contains(strings.ToLower(target.Name), "wsgi") {
			wsTargets = append(wsTargets, target)
		}
	}

	p := plot.New()
	p.Title.Text = fmt.Sprintf("%s (Workers: %d)", metricName, subConfig.NumWorkers)
	p.Y.Label.Text = yLabel

	grid := plotter.NewGrid()
	grid.Vertical.Color = color.RGBA{R: 220, G: 220, B: 220, A: 255}
	grid.Horizontal.Color = color.RGBA{R: 220, G: 220, B: 220, A: 255}
	p.Add(grid)

	numTargets := len(wsTargets)
	barWidth := vg.Points(12)

	premiumColors := []color.RGBA{
		{R: 79, G: 70, B: 229, A: 255},  // Indigo
		{R: 13, G: 148, B: 136, A: 255}, // Teal
		{R: 219, G: 39, B: 119, A: 255}, // Pink/Rose
		{R: 217, G: 119, B: 6, A: 255},  // Amber
	}

	for i, target := range wsTargets {
		values := make(plotter.Values, len(wsStages))
		for j, stage := range wsStages {
			stageName := fmt.Sprintf("WS_%s_req_%s_resp", stage.client, stage.server)
			config := Config{
				Target:     target,
				Duration:   subConfig.Duration,
				Cooldown:   subConfig.Cooldown,
				NumWorkers: subConfig.NumWorkers,
			}
			stats := wsResults[stageName][config]
			var val float64
			switch metricName {
			case "Average RPS":
				val = stats.AvgRPS
			case "Average Latency (ms)":
				val = float64(stats.AvgLatency.Milliseconds())
			default:
				val = 0
			}
			values[j] = val
		}

		bars, err := plotter.NewBarChart(values, barWidth)
		if err != nil {
			return err
		}

		colorIdx := i % len(premiumColors)
		bars.Color = premiumColors[colorIdx]
		bars.LineStyle.Color = premiumColors[colorIdx]
		bars.LineStyle.Width = vg.Points(1)

		offset := (float64(i) - float64(numTargets-1)/2.0) * float64(barWidth)
		bars.Offset = vg.Points(offset)

		p.Add(bars)
		p.Legend.Add(target.Name, bars)
	}

	stageNames := make([]string, len(wsStages))
	for j, stage := range wsStages {
		stageNames[j] = fmt.Sprintf("%s/%s", stage.client[:1], stage.server[:1])
	}
	p.NominalX(stageNames...)

	plotWidth := vg.Points(float64(len(wsStages)*numTargets*20 + 200))
	if plotWidth < 450 {
		plotWidth = 450
	}
	plotHeight := vg.Points(350)

	return p.Save(plotWidth, plotHeight, filename)
}

type ExportedRecord struct {
	Target   string         `json:"target"`
	URL      string         `json:"url"`
	Duration string         `json:"duration"`
	Cooldown string         `json:"cooldown"`
	Workers  int            `json:"workers"`
	Stats    BenchmarkStats `json:"stats"`
}

type ExportedWSRecord struct {
	Stage    string         `json:"stage"`
	Target   string         `json:"target"`
	URL      string         `json:"url"`
	Duration string         `json:"duration"`
	Cooldown string         `json:"cooldown"`
	Workers  int            `json:"workers"`
	Stats    BenchmarkStats `json:"stats"`
}

type ExportedData struct {
	CRUD      []ExportedRecord   `json:"crud"`
	Counter   []ExportedRecord   `json:"counter"`
	WebSocket []ExportedWSRecord `json:"websocket"`
}

func saveMetricsJSON(filename string, crudResults map[Config]BenchmarkStats, counterResults map[Config]BenchmarkStats, wsResults map[string]map[Config]BenchmarkStats) {
	var data ExportedData

	for config, stats := range crudResults {
		data.CRUD = append(data.CRUD, ExportedRecord{
			Target:   config.Target.Name,
			URL:      config.Target.URL,
			Duration: config.Duration.String(),
			Cooldown: config.Cooldown.String(),
			Workers:  config.NumWorkers,
			Stats:    stats,
		})
	}

	for config, stats := range counterResults {
		data.Counter = append(data.Counter, ExportedRecord{
			Target:   config.Target.Name,
			URL:      config.Target.URL,
			Duration: config.Duration.String(),
			Cooldown: config.Cooldown.String(),
			Workers:  config.NumWorkers,
			Stats:    stats,
		})
	}

	for stage, configsMap := range wsResults {
		for config, stats := range configsMap {
			data.WebSocket = append(data.WebSocket, ExportedWSRecord{
				Stage:    stage,
				Target:   config.Target.Name,
				URL:      config.Target.URL,
				Duration: config.Duration.String(),
				Cooldown: config.Cooldown.String(),
				Workers:  config.NumWorkers,
				Stats:    stats,
			})
		}
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("Error marshalling metrics to JSON: %v", err)
		return
	}

	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		log.Printf("Error writing metrics JSON file: %v", err)
		return
	}
	log.Printf("Saved metrics JSON to: %s", filename)
}
