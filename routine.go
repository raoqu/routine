package main

import (
	"context"
	"sync/atomic"
)

// RoutineControl manages the execution of a routine.
// It is parameterized by TConfig, the type of its configuration, and
// TOutput, the type of its result.
type RoutineControl[TConfig any, TOutput any] struct {
	Cancel context.CancelFunc
	Done   chan struct{}
	Output atomic.Value // Stores TOutput
	Config atomic.Value // Stores TConfig
}

// NewRoutineControl creates a new RoutineControl.
func NewRoutineControl[TConfig any, TOutput any](config TConfig, initOutput TOutput) *RoutineControl[TConfig, TOutput] {

	_, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	ctrl := &RoutineControl[TConfig, TOutput]{
		Cancel: cancel,
		Done:   done,
	}
	ctrl.Config.Store(config)
	ctrl.Output.Store(initOutput)
	return ctrl
}

// Generic function types for a Routine
type RoutineJob[TConfig any, TOutput any] func(ctrl *RoutineControl[TConfig, TOutput]) (TOutput, error)
type RoutineIdentity[TConfig any] func(config TConfig) string
type RoutineStatus func() string

// Serialization/deserialization function types
type ConfigSerializer[TConfig any] func(config TConfig) string
type ConfigDeserializer[TConfig any] func(configStr string) (TConfig, error)
type OutputSerializer[TOutput any] func(output TOutput) string

// Routine is a generic struct that represents a job to be executed.
// It is parameterized by TConfig, the type of its configuration, and
// TOutput, the type of its result.
type Routine[TConfig any, TOutput any] struct {
	Job         RoutineJob[TConfig, TOutput]
	GenIdentity RoutineIdentity[TConfig]
	GetStatus   RoutineStatus
	// Serialization/deserialization functions
	SerializeConfig   ConfigSerializer[TConfig]
	DeserializeConfig ConfigDeserializer[TConfig]
	SerializeOutput   OutputSerializer[TOutput]
}

// RoutineCreateFunc is a generic function type for creating routines.
// It takes a configuration object and returns a new routine instance.
type RoutineCreateFunc[TConfig any, TOutput any] func(config TConfig) *Routine[TConfig, TOutput]
