# Benchmarking Suite Methodology & Architecture

This directory contains the automated performance benchmarking harness and framework implementations for evaluating **Lariv** alongside other web frameworks and ERP engines (**Django ASGI**, **Django WSGI**, **Odoo**, and **Frappe**).

The benchmark suite is implemented in Go (`main.go`) for maximum throughput, low measurement overhead, and high-concurrency client simulation.

---

## Table of Contents

1. [Architecture & Harness Design](#architecture--harness-design)
2. [Tested Targets & Framework Matrix](#tested-targets--framework-matrix)
3. [Benchmark Workflows & Methodologies](#benchmark-workflows--methodologies)
   - [1. CRUD Benchmark (`crud`)](#1-crud-benchmark-crud)
   - [2. Counter Benchmark (`counter`)](#2-counter-benchmark-counter)
   - [3. Background Task Benchmark (`task`)](#3-background-task-benchmark-task)
   - [4. WebSocket Benchmark (`websocket`)](#4-websocket-benchmark-websocket)
4. [Metrics & Measurement Methodology](#metrics--measurement-methodology)
5. [Database Isolation & Cleanup](#database-isolation--cleanup)
6. [CLI & Execution Options](#cli--execution-options)

---

## Architecture & Harness Design

The benchmarking runner (`main.go`) is built using [fasthttp](https://github.com/valyala/fasthttp) for HTTP benchmarks and `golang.org/x/net/websocket` for WebSocket benchmarks.

### Key Harness Features

- **Concurrent Worker Pools**: A configurable number of concurrent worker goroutines (`workers = [1, 50, 500]`) execute test cycles in parallel against target backends.
- **TCP Connection Monitoring (`watchedConn`)**: Wraps standard `net.Conn` connections to monitor active TCP sockets dynamically. A background sampler goroutine ticks every 10 milliseconds to record average and peak active TCP connection counts.
- **RPS Bucketing**: Request-per-second (RPS) counts are grouped into 1-second time buckets to compute exact `Max RPS` spikes alongside `Average RPS`.
- **Payload Decompression & Wire Measurement**: Accurate calculation of wire data transfer (HTTP headers + body). For Gzip-compressed responses, bytes sent/received reflect wire footprint while response bodies are uncompressed in memory for payload validation.

---

## Tested Targets & Framework Matrix

Targets are configured via `config.toml` and mapped to supported benchmark types:

| Target | Port / Protocol | Supported Benchmarks |
| :--- | :--- | :--- |
| **Lariv Local Server** | `http://localhost:8123` | `crud`, `counter`, `task`, `websocket` |
| **Django Local Server (ASGI)** | `http://localhost:8124` | `crud`, `counter`, `task`, `websocket` |
| **Django Local Server (WSGI)** | `http://localhost:8125` | `crud`, `counter`, `task` *(WSGI excludes WebSockets)* |
| **Odoo Local Server** | `http://localhost:8126` | `crud`, `counter`, `task`, `websocket` |
| **Frappe Local Server** | `http://localhost:8127` | `crud`, `counter`, `task`, `websocket` |

---

## Benchmark Workflows & Methodologies

### 1. CRUD Benchmark (`crud`)

* **Endpoint**: `/api/articles/` (List/Create) and `/api/articles/<id>/` (Detail/Update/Delete)
* **Objective**: Evaluate database ORM read/write performance, CRUD query efficiency, JSON serialization/deserialization overhead, and HTTP request routing under concurrent load.
* **ID Pool Management (`SafeIDPool`)**:
  - Maintains a thread-safe thread pool of newly created article IDs.
  - Used by worker goroutines to select valid targets for `PUT` (updates) and `DELETE` actions.

#### Workflow Distribution
Each worker continuously selects one of 3 workflows with equal probability (`rand(1..3)`):

1. **Workflow 1: Write & Mutation Heavy** (33.3% choice):
   - **Create (1/3)**: Sends `POST /api/articles/` with payload `{"title":"W1_<rand5>","content":"Content_<rand10>"}`. On HTTP 201 Created, extracts article `id` into `SafeIDPool`.
   - **Update (1/3)**: Retrieves a random ID from the newest 50 IDs in `SafeIDPool` and sends `PUT /api/articles/<id>/` with updated title/content. Falls back to Create if pool is empty.
   - **Delete (1/3)**: If `SafeIDPool` size exceeds 100 entries, removes the oldest ID and issues `DELETE /api/articles/<id>/`. Otherwise, falls back to Create.

2. **Workflow 2: Read & Fetch Heavy** (33.3% choice):
   - **Create (1/3)**: `POST /api/articles/` with title `W2_<rand5>`.
   - **List (1/3)**: Sends `GET /api/articles/` to retrieve all stored articles.
   - **Delete (1/3)**: If pool size > 100, deletes oldest ID via `DELETE /api/articles/<id>/`. Otherwise, falls back to List (`GET`).

3. **Workflow 3: Search & Filter Heavy** (33.3% choice):
   - **Create (1/3)**: `POST /api/articles/` with title `W3_<rand5>`.
   - **Filtered Search (1/3)**: Selects a random character (`a-z`) and requests `GET /api/articles/?title=<char>` (executing a `LIKE %char%` query).
   - **Delete (1/3)**: If pool size > 100, deletes oldest ID via `DELETE /api/articles/<id>/`. Otherwise, falls back to List (`GET`).

---

### 2. Counter Benchmark (`counter`)

* **Endpoint**: `/api/counter/`
* **Objective**: Measure raw HTTP framework routing performance, CPU overhead, and JSON parsing/encoding speed without database I/O.
* **Methodology**:
  - Worker goroutines send HTTP `POST` requests containing JSON payload `{"counter": 42}`.
  - The server decodes JSON, increments the value by 1 (`43`), and responds with JSON `{"counter": 43}`.
  - Validates framework overhead under pure CPU-bound micro-benchmark conditions.

---

### 3. Background Task Benchmark (`task`)

* **Endpoints**:
  - Submission: `POST /api/task/`
  - Status Polling: `GET /api/task/<task_id>/`
* **Objective**: Test background task scheduling, asynchronous job queue execution (e.g. Celery, async worker routines), non-blocking request handling, and status polling efficiency.
* **Methodology**:
  - Each worker goroutine executes a complete 2-phase lifecycle per iteration:
    1. **Task Submission**: Worker sends `POST /api/task/` with plain text integer body `"42"`. Target server enqueues a background job to increment the number and immediately returns a unique string `task_id`.
    2. **Status Polling**: Worker enters a non-blocking loop issuing `GET /api/task/<task_id>/` until the job reports completion.
  - **Completion Criterion**: The cycle is complete when status equals `"completed"` with calculated `result: 43`.
  - Latency measurement spans the total duration from initial task submission to final completed status polling response.

---

### 4. WebSocket Benchmark (`websocket`)

* **Endpoint**: `/api/ws/`
* **Objective**: Measure full-duplex WebSocket connection handling, frame encoding/decoding, socket message serialization, and throughput scalability across small, medium, and large payloads.
* **Payload Matrix (9 Stages)**:
  - Executes benchmarks across a 3x3 matrix of client request and server response payload sizes:
    - `small` request / `small` response
    - `small` request / `medium` response
    - `small` request / `large` response
    - `medium` request / `small` response
    - `medium` request / `medium` response
    - `medium` request / `large` response
    - `large` request / `small` response
    - `large` request / `medium` response
    - `large` request / `large` response

#### Payload Definitions
* **Small Payload (~100 Bytes)**: Minimal JSON ping event (`{"type":"ping","timestamp":1783993200123,"client_id":"client_8b31a","seq":1024,"payload":"hello"}`).
* **Medium Payload (~3.5 KB)**: Dashboard metrics payload containing session metadata and an array of 50 metric indicator items.
* **Large Payload (~250 KB)**: Bulk sync payload containing 500 detailed article records, author metadata, and tag lists.

#### Methodology
- persistent WebSocket connections (`ws://<host>/api/ws/`) are opened per worker.
- Worker sends JSON frame `{ "query": "<requested_server_size>", "data": <client_payload_data> }`.
- Server decodes the frame and echoes back the requested response size (`small`, `medium`, or `large`).
- Records round-trip message latency, frames per second (RPS), connection count, and data volume.

---

## Metrics & Measurement Methodology

During each benchmark permutation, the runner collects the following metrics:

- **Total Requests**: Total completed request cycles.
- **Successful / Failed Requests**: HTTP 2xx / non-2xx status counts or socket errors.
- **Average RPS**: `Total Requests / Benchmark Duration (seconds)`.
- **Max RPS**: Peak request count observed within any single 1-second interval bucket.
- **Average Latency**: Average elapsed time per request cycle.
- **Max Latency**: Worst-case single request latency observed during the run.
- **Average & Max Connections**: Mean and peak active TCP sockets tracked via 10ms sampling interval.
- **Bytes Sent / Received**: Total, average, and maximum bytes transferred (including HTTP headers and raw wire payload).

All metrics are automatically dumped to `benchmark_metrics.json` and converted into interactive SVG comparison plots (`average_rps.svg`, `average_latency.svg`, `websocket_average_rps_w50.svg`, etc.).

---

## Database Isolation & Cleanup

To prevent table bloated size from skewing query latencies across iterations:
- After every CRUD benchmark run, the harness waits 1 second and executes an HTTP `POST` to `/api/truncate/`.
- Target backends issue `TRUNCATE TABLE articles RESTART IDENTITY CASCADE` (or equivalent framework table reset) to clear stored records and reset auto-increment counters before the next test permutation begins.

---

## CLI & Execution Options

Run all benchmarks:
```bash
go run main.go -all
```

Run specific benchmark suites:
```bash
go run main.go -crud
go run main.go -counter
go run main.go -task
go run main.go -ws
```

Custom configuration file:
```bash
go run main.go -config config.toml -all
```
