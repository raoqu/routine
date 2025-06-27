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
	configStr := r.URL.Query().Get("config")
	count, _ := strconv.Atoi(countStr)

	var result *HandleResult = NewHandleResult(count, "Failed to start all requested routines")

	started := 0

	// If count is 0 or negative, return an error
	if count <= 0 {
		result.SetError("Invalid count parameter: must be greater than 0").Response(w)
		return
	}

	// If config is required but not provided, return an error
	if configStr == "" {
		result.SetError("Config parameter is required but was not provided").Response(w)
		return
	}

	// Validate config before starting any routines
	_, err := s.Routine.DeserializeConfig(configStr)
	if err != nil {
		result.SetError(fmt.Sprintf("Invalid config: %v", err)).Response(w)
		return
	}

	for i := 0; i < count; i++ {
		id := fmt.Sprintf("worker-%d", time.Now().UnixNano())

		// Use a function to properly scope the recovery for each iteration
		func() {
			defer func() {
				if r := recover(); r != nil {
					result.SetError(fmt.Sprintf("Some routines failed to start: %v", r))
				}
			}()

			// Attempt to deserialize and use the config
			config, err := s.Routine.DeserializeConfig(configStr)
			if err != nil {
				result.SetError(fmt.Sprintf("Failed to deserialize config: %v", err))
				return
			}
			s.startRoutineWithConfig(id, config)
			started++
		}()
	}

	result.Set(started, count).Response(w)
}

// handleStop stops routines based on request body
func (s *RoutineScheduler[TConfig, TOutput]) handleStop(w http.ResponseWriter, r *http.Request) {
	var ids []string
	var result *HandleResult = NewHandleResult(len(ids), "Failed to stop all requested routines")
	if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
		result.SetError(fmt.Sprintf("Invalid request format: %v", err)).Response(w)
		return
	}

	stopped := 0

	for _, id := range ids {
		if _, ok := routineMap.Load(id); ok {
			s.stopRoutine(id)
			stopped++
		} else {
			result.SetError(fmt.Sprintf("Routine %s not found", id)).Response(w)
		}
	}

	result.Set(stopped, len(ids)).Response(w)
}

// handleUpdateConfig updates routine configs based on request body
func (s *RoutineScheduler[TConfig, TOutput]) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	type UpdateConfigPayload struct {
		IDs    []string `json:"ids"`
		Config string   `json:"config"`
	}

	var payload UpdateConfigPayload
	var result *HandleResult = NewHandleResult(0, "Failed to update all requested routines")

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		result.SetError(fmt.Sprintf("Invalid request format: %v", err)).Response(w)
		return
	}

	// Track success and errors
	updated := 0

	// Use the routine instance from the scheduler
	routine := s.Routine

	for _, id := range payload.IDs {
		if val, ok := routineMap.Load(id); ok {
			// Type assertion to get the control object
			ctrl, ok := val.(*RoutineControl[TConfig, TOutput])
			if !ok {
				log.Printf("Error: could not convert routine %s to expected type", id)
				result.SetError(fmt.Sprintf("Could not convert routine %s to expected type", id))
				continue
			}

			// Deserialize the config string to a config object
			if payload.Config != "" {
				newConfig, err := routine.DeserializeConfig(payload.Config)
				if err != nil {
					log.Printf("Error: could not deserialize config for routine %s: %v", id, err)
					result.SetError(fmt.Sprintf("Could not deserialize config for routine %s: %v", id, err))
					continue
				}
				ctrl.Config.Store(newConfig)
				updated++
			} else {
				// No config provided
				result.SetError("No config provided")
			}
		} else {
			result.SetError(fmt.Sprintf("Routine %s not found", id))
		}
	}

	result.Set(updated, len(payload.IDs)).Response(w)
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

type HandleResult struct {
	Success             bool   `json:"success"`
	Error               string `json:"error"`
	SuccessCount        int    `json:"success_count"`
	TotalCount          int    `json:"total_count"`
	DefaultErrorMessage string `json:"-"`
}

func NewHandleResult(totalCount int, defaultErrorMessage string) *HandleResult {
	return &HandleResult{
		Success:             false,
		Error:               "",
		SuccessCount:        0,
		TotalCount:          totalCount,
		DefaultErrorMessage: defaultErrorMessage,
	}
}

func (result *HandleResult) SetSuccess() *HandleResult {
	result.Success = true
	result.Error = ""
	return result
}

func (result *HandleResult) SetError(error string) *HandleResult {
	result.Success = false
	result.Error = error
	return result
}

func (result *HandleResult) Set(successCount, totalCount int) *HandleResult {
	result.SuccessCount = successCount
	result.TotalCount = totalCount
	return result
}

func (result *HandleResult) Response(w http.ResponseWriter) {
	result.Success = (result.SuccessCount == result.TotalCount) && (result.Error == "")
	if (result.SuccessCount != result.TotalCount) && (result.Error != "") {
		result.Error = result.DefaultErrorMessage
	}

	w.Header().Set("Content-Type", "application/json")
	if result.Success {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
	json.NewEncoder(w).Encode(result)
}
