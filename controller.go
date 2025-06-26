package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Global scheduler instance to access from handlers
var scheduler *RoutineScheduler

func handleStart(w http.ResponseWriter, r *http.Request) {
	// Get count parameter
	countStr := r.URL.Query().Get("count")
	count, _ := strconv.Atoi(countStr)
	
	// Get config parameter (optional)
	configStr := r.URL.Query().Get("config")
	
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("worker-%d", time.Now().UnixNano())
		
		// Start routine with config if provided
		if configStr != "" {
			scheduler.startRoutineWithConfig(id, configStr)
		} else {
			scheduler.startRoutine(id)
		}
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
		Value string  `json:"value"`
	}
	_ = json.NewDecoder(r.Body).Decode(&payload)
	
	// Update routines with the string config value
	for _, id := range payload.IDs {
		if val, ok := routineMap.Load(id); ok {
			// Use type switch to handle different routine control types
			switch ctrl := val.(type) {
			case interface{ 
				GetRoutine() interface{} 
				UpdateConfig(configStr string) 
				ResetOutput() 
			}:
				// Use the interface methods to update config and reset output
				ctrl.UpdateConfig(payload.Value)
				ctrl.ResetOutput()
				
			case *RoutineControl[CustomizedConfig, CustomizedOutput]:
				// Get the routine creator to access serialization functions
				if creator, ok := scheduler.RoutineCreator.(func() *Routine[CustomizedConfig, CustomizedOutput]); ok {
					routine := creator()
					// Deserialize the config string
					newConfig := routine.DeserializeConfig(payload.Value)
					ctrl.Config.Store(newConfig)
					// Reset output
					ctrl.Output.Store(DefaultCustomizedOutput())
				}
			}
		}
	}
	
	fmt.Fprintf(w, "Updated config for %d routines", len(payload.IDs))
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	type RoutineInfo struct {
		ID        string `json:"id"`
		OutputStr string `json:"output"`
		ConfigStr string `json:"config"`
		Type      string `json:"type"`
	}

	// Get filter parameter from query string
	filterID := r.URL.Query().Get("filter")
	
	var routines []RoutineInfo
	
	// Collect status from all routines
	routineMap.Range(func(key, val any) bool {
		// Apply ID filter if provided
		id := key.(string)
		if filterID != "" && !strings.Contains(strings.ToLower(id), strings.ToLower(filterID)) {
			return true // Skip this routine if it doesn't match the filter
		}
		
		// Use type switch to handle different routine control types
		switch ctrl := val.(type) {
		case interface{ 
			GetRoutineType() string 
			GetSerializedConfig() string 
			GetSerializedOutput() string 
		}:
			// Use the interface methods to get serialized data
			routines = append(routines, RoutineInfo{
				ID:        id,
				OutputStr: ctrl.GetSerializedOutput(),
				ConfigStr: ctrl.GetSerializedConfig(),
				Type:      ctrl.GetRoutineType(),
			})
			
		case *RoutineControl[CustomizedConfig, CustomizedOutput]:
			// Get the routine creator to access serialization functions
			if creator, ok := scheduler.RoutineCreator.(func() *Routine[CustomizedConfig, CustomizedOutput]); ok {
				routine := creator()
				output := ctrl.Output.Load().(CustomizedOutput)
				config := ctrl.Config.Load().(CustomizedConfig)
				
				routines = append(routines, RoutineInfo{
					ID:        id,
					OutputStr: routine.SerializeOutput(output),
					ConfigStr: routine.SerializeConfig(config),
					Type:      "CustomizedRoutine",
				})
			}
		}
		return true
	})

	_ = json.NewEncoder(w).Encode(routines)
}
