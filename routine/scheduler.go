// Package routine provides tools for managing concurrent routines.
package routine

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// RoutineScheduler manages the creation, execution, and termination of routines
// It is parameterized by TConfig, the type of its configuration, and
// TOutput, the type of its result.
type RoutineScheduler[TConfig, TOutput any] struct {
	Port int
	// Routine is the stateless routine definition to use for all instances
	Routine *Routine[TConfig, TOutput]
	// InteractiveMode indicates whether the application is running in interactive mode
	InteractiveMode bool
}

func (s *RoutineScheduler[TConfig, TOutput]) StopRoutines(ids []string) (int, error) {
	stopped := 0
	var err error
	for _, id := range ids {
		if _, ok := routineMap.Load(id); ok {
			errStop := s.StopRoutine(id)
			if errStop != nil {
				err = errStop
				continue
			}
			stopped++
		} else {
			err = fmt.Errorf("routine %s not found", id)
		}
	}
	return stopped, err
}

func (s *RoutineScheduler[TConfig, TOutput]) UpdateRoutineConfig(ids []string, newConfig TConfig) (int, error) {
	var err error
	updated := 0
	for _, id := range ids {
		if val, ok := routineMap.Load(id); ok {
			// Type assertion to get the control object
			ctrl, ok := val.(*RoutineControl[TConfig, TOutput])

			if !ok {
				err = fmt.Errorf("could not convert routine %s to expected type", id)
				continue
			} else {
				ctrl.Config.Store(newConfig)
				updated++
			}
		} else {
			err = fmt.Errorf("routine %s not found", id)
		}
	}
	return updated, err
}

// NewRoutineScheduler creates a new scheduler with the specified port and routine
// The routine parameter should be a pointer to a Routine instance
func NewRoutineScheduler[TConfig, TOutput any](port int, routine *Routine[TConfig, TOutput], interactiveMode bool) *RoutineScheduler[TConfig, TOutput] {
	return &RoutineScheduler[TConfig, TOutput]{
		Port:            port,
		Routine:         routine,
		InteractiveMode: interactiveMode,
	}
}

var (
	// Use type assertion when retrieving from the map
	routineMap sync.Map // map[string]interface{}
)

// startRoutineWithConfig creates and starts a new routine with the given ID and config
func (s *RoutineScheduler[TConfig, TOutput]) StartRoutineWithConfig(config TConfig) (string, error) {
	// Use the routine instance from the scheduler
	routine := s.Routine
	id := routine.GenIdentity(config)

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
		// Make sure to remove the routine from the map when the goroutine exits
		defer routineMap.Delete(id)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Execute the routine job and update the output
				newOutput, err := routine.Job(ctrl)
				if err != nil {
					log.Printf("job runtime error: %v", err)
					return
				} else {
					ctrl.Output.Store(newOutput)
				}
			}
		}
	}()
	return id, nil
}

// stopRoutine stops a running routine with the given ID
func (s *RoutineScheduler[TConfig, TOutput]) StopRoutine(id string) error {
	if val, ok := routineMap.Load(id); ok {
		// Type assertion to get the control object
		ctrl, ok := val.(*RoutineControl[TConfig, TOutput])
		if !ok {
			return fmt.Errorf("could not convert routine %s to expected type", id)
		}

		ctrl.Cancel()
	}
	return nil
}

// SuspendRoutine suspends a running routine with the given ID
func (s *RoutineScheduler[TConfig, TOutput]) SuspendRoutine(id string) error {
	if val, ok := routineMap.Load(id); ok {
		// Type assertion to get the control object
		ctrl, ok := val.(*RoutineControl[TConfig, TOutput])
		if !ok {
			return fmt.Errorf("could not convert routine %s to expected type", id)
		}

		// Call the routine's suspend function if available
		if s.Routine.Suspend != nil {
			s.Routine.Suspend(ctrl)
		}
		return nil
	}
	return fmt.Errorf("routine %s not found", id)
}

// ResumeRoutine resumes a suspended routine with the given ID
func (s *RoutineScheduler[TConfig, TOutput]) ResumeRoutine(id string) error {
	if val, ok := routineMap.Load(id); ok {
		// Type assertion to get the control object
		ctrl, ok := val.(*RoutineControl[TConfig, TOutput])
		if !ok {
			return fmt.Errorf("could not convert routine %s to expected type", id)
		}

		// Call the routine's resume function if available
		if s.Routine.Resume != nil {
			s.Routine.Resume(ctrl)
		}
		return nil
	}
	return fmt.Errorf("routine %s not found", id)
}

// SuspendRoutines suspends multiple routines with the given IDs
func (s *RoutineScheduler[TConfig, TOutput]) SuspendRoutines(ids []string) (int, error) {
	suspended := 0
	var err error
	for _, id := range ids {
		errSuspend := s.SuspendRoutine(id)
		if errSuspend != nil {
			err = errSuspend
			continue
		}
		suspended++
	}
	return suspended, err
}

// ResumeRoutines resumes multiple routines with the given IDs
func (s *RoutineScheduler[TConfig, TOutput]) ResumeRoutines(ids []string) (int, error) {
	resumed := 0
	var err error
	for _, id := range ids {
		errResume := s.ResumeRoutine(id)
		if errResume != nil {
			err = errResume
			continue
		}
		resumed++
	}
	return resumed, err
}
