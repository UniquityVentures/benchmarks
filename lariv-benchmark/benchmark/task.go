package benchmark

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
)

type TaskJob struct {
	ID  string
	Val int64
}

type TaskResult struct {
	Status string `json:"status"`
	Result int64  `json:"result,omitempty"`
}

var (
	taskChan   chan TaskJob
	taskStore  sync.Map // map[string]*TaskResult
	taskSeq    uint64
	workerOnce sync.Once
)

func initWorkerPool() {
	workerOnce.Do(func() {
		taskChan = make(chan TaskJob, 500000)
		numWorkers := 32
		for i := 0; i < numWorkers; i++ {
			go func() {
				for job := range taskChan {
					res := &TaskResult{
						Status: "completed",
						Result: job.Val + 1,
					}
					taskStore.Store(job.ID, res)
				}
			}()
		}
	})
}

func ResetTaskStore() {
	taskStore.Range(func(key, value any) bool {
		taskStore.Delete(key)
		return true
	})
}

// TaskSubmitHandler handles POST /api/task/
func TaskSubmitHandler(w http.ResponseWriter, r *http.Request) {
	initWorkerPool()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	val, err := strconv.ParseInt(string(body), 10, 64)
	if err != nil {
		http.Error(w, "invalid integer payload", http.StatusBadRequest)
		return
	}

	idNum := atomic.AddUint64(&taskSeq, 1)
	idStr := strconv.FormatUint(idNum, 10)

	taskStore.Store(idStr, &TaskResult{Status: "pending"})

	taskChan <- TaskJob{ID: idStr, Val: val}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(idStr))
}

// TaskStatusHandler handles GET /api/task/{id}/
func TaskStatusHandler(w http.ResponseWriter, r *http.Request) {
	initWorkerPool()

	id := r.PathValue("id")
	val, ok := taskStore.Load(id)
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	res := val.(*TaskResult)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}
