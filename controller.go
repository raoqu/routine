package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Global scheduler instance to access from handlers
var scheduler *RoutineScheduler

func handleStart(w http.ResponseWriter, r *http.Request) {
	countStr := r.URL.Query().Get("count")
	count, _ := strconv.Atoi(countStr)
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("worker-%d", time.Now().UnixNano())
		scheduler.startRoutine(id)
	}
	fmt.Fprintf(w, "Started %d routines", count)
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	var ids []string
	_ = json.NewDecoder(r.Body).Decode(&ids)
	for _, id := range ids {
		scheduler.stopRoutine(id)
	}
	fmt.Fprintf(w, "Stopped %d routines", len(ids))
}

func handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		IDs   []string `json:"ids"`
		Value int      `json:"value"`
	}
	_ = json.NewDecoder(r.Body).Decode(&payload)
	for _, id := range payload.IDs {
		if val, ok := routineMap.Load(id); ok {
			ctrl := val.(*RoutineControl[int, int])
			ctrl.Config.Store(payload.Value)
			ctrl.Output.Store(0)
		}
	}
	fmt.Fprintf(w, "Updated config for %d routines", len(payload.IDs))
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	type RoutineInfo struct {
		ID     string `json:"id"`
		Output int    `json:"output"`
		Config int    `json:"config"`
	}

	var routines []RoutineInfo
	routineMap.Range(func(key, val any) bool {
		ctrl := val.(*RoutineControl[int, int])
		routines = append(routines, RoutineInfo{
			ID:     key.(string),
			Output: ctrl.Output.Load().(int),
			Config: ctrl.Config.Load().(int),
		})
		return true
	})

	_ = json.NewEncoder(w).Encode(routines)
}
