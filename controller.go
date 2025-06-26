package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Handler to check if the application is in test mode
func (s *RoutineScheduler[TConfig, TOutput]) handleTestMode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"testMode": IsTestMode})
}

// Handler to switch test mode on or off
func (s *RoutineScheduler[TConfig, TOutput]) handleSwitchTestMode(w http.ResponseWriter, r *http.Request) {
	// Get the desired mode from the query parameter
	mode := r.URL.Query().Get("mode")
	
	if mode == "on" {
		IsTestMode = true
		log.Println("Switched to test mode")
	} else if mode == "off" {
		IsTestMode = false
		log.Println("Switched to normal mode")
	}
	
	// Return the current mode
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"testMode": IsTestMode})
}

// handleHome serves the main HTML page
func (s *RoutineScheduler[TConfig, TOutput]) handleHome(w http.ResponseWriter, r *http.Request) {
	// Always serve the index.html file directly
	// The JavaScript in the HTML will check for the isTestMode flag
	http.ServeFile(w, r, "index.html")
}

// handleStart starts new routines based on request parameters
func (s *RoutineScheduler[TConfig, TOutput]) handleStart(w http.ResponseWriter, r *http.Request) {
	// Get count parameter
	countStr := r.URL.Query().Get("count")
	count, _ := strconv.Atoi(countStr)

	// Get config parameter (optional)
	configStr := r.URL.Query().Get("config")

	for i := 0; i < count; i++ {
		id := fmt.Sprintf("worker-%d", time.Now().UnixNano())

		// Start routine with config if provided
		if configStr != "" {
			// Deserialize the config string to a TConfig object
			config := s.Routine.DeserializeConfig(configStr)
			s.startRoutineWithConfig(id, config)
		} else {
			s.startRoutine(id)
		}
	}

	fmt.Fprintf(w, "Started %d routines", count)
}

// handleStop stops routines based on request body
func (s *RoutineScheduler[TConfig, TOutput]) handleStop(w http.ResponseWriter, r *http.Request) {
	var ids []string
	_ = json.NewDecoder(r.Body).Decode(&ids)
	for _, id := range ids {
		s.stopRoutine(id)
	}
	fmt.Fprintf(w, "Stopped %d routines", len(ids))
}

// handleUpdateConfig updates routine configs based on request body
func (s *RoutineScheduler[TConfig, TOutput]) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	type UpdateConfigPayload struct {
		IDs    []string `json:"ids"`
		Config string   `json:"config"`
	}

	var payload UpdateConfigPayload
	_ = json.NewDecoder(r.Body).Decode(&payload)

	// Use the routine instance from the scheduler
	routine := s.Routine

	for _, id := range payload.IDs {
		if val, ok := routineMap.Load(id); ok {
			// Type assertion to get the control object
			ctrl, ok := val.(*RoutineControl[TConfig, TOutput])
			if !ok {
				log.Printf("Error: could not convert routine %s to expected type", id)
				continue
			}

			// Deserialize the config string to a config object
			if payload.Config != "" {
				newConfig := routine.DeserializeConfig(payload.Config)
				ctrl.Config.Store(newConfig)
			} else {
				// Use default config if none provided
				var defaultConfig TConfig
				ctrl.Config.Store(defaultConfig)
			}
		}
	}

	fmt.Fprintf(w, "Updated config for %d routines", len(payload.IDs))
}

// handleStatus returns the status of all routines
func (s *RoutineScheduler[TConfig, TOutput]) handleStatus(w http.ResponseWriter, r *http.Request) {
	type RoutineInfo struct {
		ID        string `json:"id"`
		OutputStr string `json:"output"`
		ConfigStr string `json:"config"`
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
		case interface {
			GetRoutineType() string
			GetSerializedConfig() string
			GetSerializedOutput() string
		}:
			// Use the interface methods to get serialized data
			routines = append(routines, RoutineInfo{
				ID:        id,
				OutputStr: ctrl.GetSerializedOutput(),
				ConfigStr: ctrl.GetSerializedConfig(),
			})

		case *RoutineControl[TConfig, TOutput]:
			// Use the routine instance from the scheduler
			routine := s.Routine
			output := ctrl.Output.Load().(TOutput)
			config := ctrl.Config.Load().(TConfig)

			routines = append(routines, RoutineInfo{
				ID:        id,
				OutputStr: routine.SerializeOutput(output),
				ConfigStr: routine.SerializeConfig(config),
			})
		}
		return true
	})

	_ = json.NewEncoder(w).Encode(routines)
}

func (s *RoutineScheduler[TConfig, TOutput]) Serve() {
	// Create a new ServeMux for this scheduler instance
	mux := http.NewServeMux()

	// Register handlers for this scheduler instance
	mux.HandleFunc("/", s.handleHome)
	mux.HandleFunc("/start", s.handleStart)
	mux.HandleFunc("/stop", s.handleStop)
	mux.HandleFunc("/update-config", s.handleUpdateConfig)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/test_mode", s.handleTestMode)
	mux.HandleFunc("/switch", s.handleSwitchTestMode)

	log.Printf("Routine server starting on port %d...", s.Port)
	err := http.ListenAndServe(":"+strconv.Itoa(s.Port), mux)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
