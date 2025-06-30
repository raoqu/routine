package main

import (
	"encoding/json"
	"fmt"
	"main/routine"
	"time"
)

// CustomizedConfig holds the configuration for a CustomizedRoutine
type CustomizedConfig struct {
	Value   int  `json:"value"`
	Suspend bool `json:"suspend"`
}

// CustomizedOutput holds the output data for a CustomizedRoutine
type CustomizedOutput struct {
	Count     int
	Timestamp time.Time
}

// CustomizedRoutine implements the Routine interface with CustomizedConfig and CustomizedOutput types
type CustomizedRoutine struct {
}

// Job implements the routine job function for CustomizedRoutine
func (r *CustomizedRoutine) Job(ctrl *routine.RoutineControl[*CustomizedConfig, *CustomizedOutput]) (*CustomizedOutput, error) {
	// If routine is suspended, return current output without changes
	config := ctrl.Config.Load().(*CustomizedConfig)
	if config.Suspend {
		if prevOutput, ok := ctrl.Output.Load().(*CustomizedOutput); ok {
			return prevOutput, nil
		}
		// If no previous output, return empty output
		return &CustomizedOutput{Timestamp: time.Now()}, nil
	}

	// Safely handle the previous output, which might be nil on first run
	var prevCount int
	if prevOutput, ok := ctrl.Output.Load().(*CustomizedOutput); ok && prevOutput != nil {
		prevCount = prevOutput.Count
	}

	// Create new output with incremented count
	newOutput := CustomizedOutput{
		Count:     prevCount + config.Value,
		Timestamp: time.Now(),
	}
	time.Sleep(100 * time.Millisecond)

	return &newOutput, nil
}

// GenIdentity implements the identity generation for CustomizedRoutine
func (r *CustomizedRoutine) GenIdentity(config *CustomizedConfig) string {
	// Generate ID with nano timestamp and config value
	return fmt.Sprintf("CR-%d-%d", time.Now().UnixNano(), config.Value)
}

// SerializeConfig implements config serialization for CustomizedRoutine
func (r *CustomizedRoutine) SerializeConfig(config *CustomizedConfig) string {
	if config == nil {
		return ""
	}
	return fmt.Sprintf("%d", config.Value)
}

// DeserializeConfig implements config deserialization for CustomizedRoutine
func (r *CustomizedRoutine) DeserializeConfig(configStr string) (*CustomizedConfig, error) {
	var cfg CustomizedConfig
	if err := json.Unmarshal([]byte(configStr), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SerializeOutput implements output serialization for CustomizedRoutine
func (r *CustomizedRoutine) SerializeOutput(output *CustomizedOutput) string {
	if output == nil {
		return ""
	}
	return fmt.Sprintf("%d\n%s",
		output.Count,
		output.Timestamp.Format(time.RFC3339))
}

// Suspend implements suspend for CustomizedRoutine
func (r *CustomizedRoutine) Suspend(ctrl *routine.RoutineControl[*CustomizedConfig, *CustomizedOutput]) {
	config := ctrl.Config.Load().(*CustomizedConfig)
	newConfig := CustomizedConfig{
		Value:   config.Value,
		Suspend: true,
	}
	ctrl.Config.Store(&newConfig)
}

// Resume implements resume for CustomizedRoutine
func (r *CustomizedRoutine) Resume(ctrl *routine.RoutineControl[*CustomizedConfig, *CustomizedOutput]) {
	config := ctrl.Config.Load().(*CustomizedConfig)
	newConfig := CustomizedConfig{
		Value:   config.Value,
		Suspend: false,
	}
	ctrl.Config.Store(&newConfig)
}

// NewCustomizedRoutine creates a new CustomizedRoutine instance
func NewCustomizedRoutine() *routine.Routine[*CustomizedConfig, *CustomizedOutput] {
	customized := &CustomizedRoutine{}

	// Convert the CustomizedRoutine to a Routine
	return &routine.Routine[*CustomizedConfig, *CustomizedOutput]{
		Job:               customized.Job,
		GenIdentity:       customized.GenIdentity,
		SerializeConfig:   customized.SerializeConfig,
		DeserializeConfig: customized.DeserializeConfig,
		SerializeOutput:   customized.SerializeOutput,
		Suspend:           customized.Suspend,
		Resume:            customized.Resume,
	}
}
