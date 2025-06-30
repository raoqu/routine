package routine

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func (s *RoutineScheduler[TConfig, TOutput]) Serve() {
	// Create a new ServeMux for this scheduler instance
	mux := http.NewServeMux()

	// Register handlers for this scheduler instance
	mux.HandleFunc("/", s.handleHome)
	mux.HandleFunc("/start", s.handleStart)
	mux.HandleFunc("/stop", s.handleStop)
	mux.HandleFunc("/suspend", s.handleSuspend)
	mux.HandleFunc("/resume", s.handleResume)
	mux.HandleFunc("/update-config", s.handleUpdateConfig)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/interactive_mode", s.handleInteractiveMode)
	mux.HandleFunc("/switch", s.handleSwitchInteractiveMode)

	log.Printf("Routine server starting on port %d...", s.Port)
	err := http.ListenAndServe(":"+strconv.Itoa(s.Port), mux)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// Handler to check if the application is in interactive mode
func (s *RoutineScheduler[TConfig, TOutput]) handleInteractiveMode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"interactiveMode": s.InteractiveMode})
}

// Handler to switch interactive mode on or off
func (s *RoutineScheduler[TConfig, TOutput]) handleSwitchInteractiveMode(w http.ResponseWriter, r *http.Request) {
	// Get the desired mode from the query parameter
	mode := r.URL.Query().Get("mode")

	if mode == "on" {
		s.InteractiveMode = true
		log.Println("Switched to test mode (non-interactive)")
	} else if mode == "off" {
		s.InteractiveMode = false
		log.Println("Switched to normal mode (interactive)")
	}

	// Return the current mode
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"interactiveMode": s.InteractiveMode})
}

// handleHome serves the main HTML page
func (s *RoutineScheduler[TConfig, TOutput]) handleHome(w http.ResponseWriter, r *http.Request) {
	// Always serve the index.html file directly
	// The JavaScript in the HTML will check for the isTestMode flag
	http.ServeFile(w, r, "static/index.html")
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
		result.SetError(errors.New("invalid count parameter: must be greater than 0")).Response(w)
		return
	}

	// If config is required but not provided, return an error
	if configStr == "" {
		result.SetError(errors.New("config parameter is required but was not provided")).Response(w)
		return
	}

	// Validate config before starting any routines
	_, err := s.Routine.DeserializeConfig(configStr)
	if err != nil {
		result.SetError(fmt.Errorf("invalid config: %v", err)).Response(w)
		return
	}

	for i := 0; i < count; i++ {
		// Use a function to properly scope the recovery for each iteration
		func() {
			defer func() {
				if r := recover(); r != nil {
					result.SetError(fmt.Errorf("some routines failed to start: %v", r))
				}
			}()

			// Attempt to deserialize and use the config
			config, err := s.Routine.DeserializeConfig(configStr)
			if err != nil {
				result.SetError(fmt.Errorf("failed to deserialize config: %v", err))
				return
			}
			id, err := s.StartRoutineWithConfig(config)
			if err != nil {
				result.SetError(fmt.Errorf("failed to start routine: %v", err))
				return
			} else if id != "" {
				started++
			}
		}()
	}

	result.Set(started, count).Response(w)
}

// handleStop stops routines based on request body
func (s *RoutineScheduler[TConfig, TOutput]) handleStop(w http.ResponseWriter, r *http.Request) {
	var ids []string
	var result *HandleResult = NewHandleResult(len(ids), "Failed to stop all requested routines")
	if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
		result.SetError(fmt.Errorf("invalid request format: %v", err)).Response(w)
		return
	}

	stopped := 0

	stopped, err := s.StopRoutines(ids)

	result.SetError(err)
	result.Set(stopped, len(ids)).Response(w)
}

// handleSuspend suspends routines based on request body
func (s *RoutineScheduler[TConfig, TOutput]) handleSuspend(w http.ResponseWriter, r *http.Request) {
	var ids []string
	var result *HandleResult = NewHandleResult(len(ids), "Failed to suspend all requested routines")
	if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
		result.SetError(fmt.Errorf("invalid request format: %v", err)).Response(w)
		return
	}

	suspended := 0

	suspended, err := s.SuspendRoutines(ids)

	result.SetError(err)
	result.Set(suspended, len(ids)).Response(w)
}

// handleResume resumes routines based on request body
func (s *RoutineScheduler[TConfig, TOutput]) handleResume(w http.ResponseWriter, r *http.Request) {
	var ids []string
	var result *HandleResult = NewHandleResult(len(ids), "Failed to resume all requested routines")
	if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
		result.SetError(fmt.Errorf("invalid request format: %v", err)).Response(w)
		return
	}

	resumed := 0

	resumed, err := s.ResumeRoutines(ids)

	result.SetError(err)
	result.Set(resumed, len(ids)).Response(w)
}

// handleUpdateConfig updates routine configs based on request body
func (s *RoutineScheduler[TConfig, TOutput]) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	type UpdateConfigPayload struct {
		IDs    []string `json:"ids"`
		Config string   `json:"config"`
	}

	var payload UpdateConfigPayload
	var result *HandleResult = NewHandleResult(0, "Failed to update all requested routines")
	result.Set(0, len(payload.IDs))

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		result.SetError(fmt.Errorf("invalid request format: %v", err)).Response(w)
		return
	}

	routine := s.Routine

	newConfig, err := routine.DeserializeConfig(payload.Config)
	if err != nil {
		log.Printf("Error: could not deserialize config %v", err)
		result.SetError(fmt.Errorf("could not deserialize config %v", err)).Response(w)
		return
	}

	updated, err := s.UpdateRoutineConfig(payload.IDs, newConfig)
	if err != nil {
		log.Printf("Error: could not update config %v", err)
		result.SetError(fmt.Errorf("could not update config %v", err)).Response(w)
		return
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

func (result *HandleResult) SetError(err error) *HandleResult {
	if err != nil {
		result.Success = false
		result.Error = err.Error()
	} else {
		result.Success = true
		result.Error = ""
	}
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
