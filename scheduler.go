// Package routine provides tools for managing concurrent routines.
package main

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RoutineScheduler manages the creation, execution, and termination of routines
type RoutineScheduler struct {
	Port int
	// RoutineCreator is a function that creates a new routine
	// It's parameterized by the config and output types
	RoutineCreator interface{}
}

// NewRoutineScheduler creates a new scheduler with the specified port and routine creator
// The routineCreator parameter should be a function that creates a new routine
// of the form func() *Routine[TConfig, TOutput]
func NewRoutineScheduler[TConfig, TOutput any](port int, routineCreator func() *Routine[TConfig, TOutput]) *RoutineScheduler {
	return &RoutineScheduler{
		Port:          port,
		RoutineCreator: routineCreator,
	}
}

func (s *RoutineScheduler) Serve() {
	// Add handlers for the web interface
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/start", handleStart)
	http.HandleFunc("/stop", handleStop)
	http.HandleFunc("/update-config", handleUpdateConfig)
	http.HandleFunc("/status", handleStatus)

	log.Printf("Server starting on port %d...", s.Port)
	err := http.ListenAndServe(":"+strconv.Itoa(s.Port), nil)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

var (
	// Use type assertion when retrieving from the map
	routineMap sync.Map // map[string]interface{}
)

// startRoutine creates and starts a new routine with the given ID using default config
func (s *RoutineScheduler) startRoutine(id string) {
	// Call startRoutineWithConfig with empty config string to use defaults
	s.startRoutineWithConfig(id, "")
}

// startRoutineWithConfig creates and starts a new routine with the given ID and config
func (s *RoutineScheduler) startRoutineWithConfig(id string, configStr string) {
	// Use type assertion to get the appropriate routine creator function
	switch creator := s.RoutineCreator.(type) {
	case func() *Routine[CustomizedConfig, CustomizedOutput]:
		// Create the routine using the creator function
		routine := creator()
		
		// Initialize config - either from provided string or default
		var config CustomizedConfig
		if configStr != "" {
			// Use the routine's deserializer to parse the config string
			config = routine.DeserializeConfig(configStr)
			log.Printf("Starting routine %s with custom config: %s", id, configStr)
		} else {
			config = DefaultCustomizedConfig()
			log.Printf("Starting routine %s with default config", id)
		}
		
		// Initialize the control with the config and default output
		ctrl := NewRoutineControl(config, DefaultCustomizedOutput())
		
		// Store the control in the map
		routineMap.Store(id, ctrl)
		
		// Create context and channels
		ctx, cancel := context.WithCancel(context.Background())
		ctrl.Cancel = cancel
		done := make(chan struct{})
		ctrl.Done = done
		
		go func() {
			defer close(done)
			log.Printf("Routine %s started", id)
			for {
				select {
				case <-ctx.Done():
					log.Printf("Routine %s exiting", id)
					return
				default:
					// Execute the routine job and update the output
					newOutput := routine.Job(ctrl)
					ctrl.Output.Store(newOutput)
					
					// Sleep between iterations
					time.Sleep(100 * time.Millisecond)
				}
			}
		}()
	
	default:
		log.Printf("Error: unsupported routine creator type")
	}
}

// stopRoutine stops a running routine with the given ID
func (s *RoutineScheduler) stopRoutine(id string) {
	if val, ok := routineMap.Load(id); ok {
		// Use type assertion based on the RoutineCreator type
		switch s.RoutineCreator.(type) {
		case func() *Routine[CustomizedConfig, CustomizedOutput]:
			ctrl, ok := val.(*RoutineControl[CustomizedConfig, CustomizedOutput])
			if !ok {
				log.Printf("Error: could not convert routine %s to CustomizedRoutine", id)
				return
			}
			
			ctrl.Cancel()
			go func() {
				<-ctrl.Done
				routineMap.Delete(id)
				log.Printf("Routine %s fully stopped", id)
			}()
			
		default:
			log.Printf("Error: unsupported routine type for stopping routine %s", id)
		}
	}
}
