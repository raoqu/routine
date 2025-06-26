// Package routine provides tools for managing concurrent routines.
package main

import (
	"context"
	"log"
	"sync"
	"time"
)

// RoutineScheduler manages the creation, execution, and termination of routines
// It is parameterized by TConfig, the type of its configuration, and
// TOutput, the type of its result.
type RoutineScheduler[TConfig, TOutput any] struct {
	Port int
	// Routine is the stateless routine definition to use for all instances
	Routine *Routine[TConfig, TOutput]
}

// NewRoutineScheduler creates a new scheduler with the specified port and routine
// The routine parameter should be a pointer to a Routine instance
func NewRoutineScheduler[TConfig, TOutput any](port int, routine *Routine[TConfig, TOutput]) *RoutineScheduler[TConfig, TOutput] {
	return &RoutineScheduler[TConfig, TOutput]{
		Port:    port,
		Routine: routine,
	}
}

var (
	// Use type assertion when retrieving from the map
	routineMap sync.Map // map[string]interface{}
)

// startRoutine creates and starts a new routine with the given ID using default config
func (s *RoutineScheduler[TConfig, TOutput]) startRoutine(id string) {
	// Create a zero value of TConfig as the default config
	defaultConfig := *new(TConfig)
	s.startRoutineWithConfig(id, defaultConfig)
}

// startRoutineWithConfig creates and starts a new routine with the given ID and config
func (s *RoutineScheduler[TConfig, TOutput]) startRoutineWithConfig(id string, config TConfig) {
	// Use the routine instance from the scheduler
	routine := s.Routine

	// Initialize the control with the config and default output
	ctrl := NewRoutineControl(config, *new(TOutput)) // Zero value for TOutput

	// Store the control in the map
	routineMap.Store(id, ctrl)

	// Create context and channels
	ctx, cancel := context.WithCancel(context.Background())
	ctrl.Cancel = cancel
	done := make(chan struct{})
	ctrl.Done = done

	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
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
}

// stopRoutine stops a running routine with the given ID
func (s *RoutineScheduler[TConfig, TOutput]) stopRoutine(id string) {
	if val, ok := routineMap.Load(id); ok {
		// Type assertion to get the control object
		ctrl, ok := val.(*RoutineControl[TConfig, TOutput])
		if !ok {
			log.Printf("Error: could not convert routine %s to expected type", id)
			return
		}

		ctrl.Cancel()
		go func() {
			<-ctrl.Done
			routineMap.Delete(id)
			log.Printf("Routine %s fully stopped", id)
		}()
	}
}
