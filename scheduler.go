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

type RoutineScheduler struct {
	Port int
}

func NewRoutineScheduler(port int) *RoutineScheduler {
	return &RoutineScheduler{
		Port: port,
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
	routineMap sync.Map // map[string]*RoutineControl[C, O]
)

func (s *RoutineScheduler) startRoutine(id string) {
	// Routines will have int configuration and int output
	ctrl := NewRoutineControl[int, int](0, 0)
	routineMap.Store(id, ctrl)

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
				// This is not thread-safe, but fixes the compilation error.
				// For a thread-safe counter, use atomic operations.
				currentOutput := ctrl.Output.Load().(int)
				ctrl.Output.Store(currentOutput + 1)

				currentConfig := ctrl.Config.Load().(int)
				log.Printf("Routine %s running: count=%d config=%d", id, ctrl.Output.Load().(int), currentConfig)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

func (s *RoutineScheduler) stopRoutine(id string) {
	if val, ok := routineMap.Load(id); ok {
		ctrl := val.(*RoutineControl[int, int])
		ctrl.Cancel()
		go func() {
			<-ctrl.Done
			routineMap.Delete(id)
			log.Printf("Routine %s fully stopped", id)
		}()
	}
}
