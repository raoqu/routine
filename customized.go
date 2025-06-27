package main

import (
	"encoding/json"
	"fmt"
	"time"
)

// CustomizedConfig holds the configuration for a CustomizedRoutine
type CustomizedConfig struct {
	Value int `json:"value"`
}

// CustomizedOutput holds the output data for a CustomizedRoutine
type CustomizedOutput struct {
	Count     int
	Timestamp time.Time
}

// CustomizedRoutine implements the Routine interface with CustomizedConfig and CustomizedOutput types
type CustomizedRoutine struct {
}

// NewCustomizedRoutine creates a new CustomizedRoutine instance
func NewCustomizedRoutine() *Routine[*CustomizedConfig, CustomizedOutput] {
	return &Routine[*CustomizedConfig, CustomizedOutput]{
		Job: func(ctrl *RoutineControl[*CustomizedConfig, CustomizedOutput]) (CustomizedOutput, error) {
			config := ctrl.Config.Load().(*CustomizedConfig)
			prevOutput := ctrl.Output.Load().(CustomizedOutput)

			// Create new output with incremented count
			newOutput := CustomizedOutput{
				Count:     prevOutput.Count + config.Value,
				Timestamp: time.Now(),
			}
			time.Sleep(100 * time.Millisecond)

			return newOutput, nil
		},
		GenIdentity: func(config *CustomizedConfig) string {
			// Generate ID with nano timestamp and config value
			return fmt.Sprintf("CR-%d-%d", time.Now().UnixNano(), config.Value)
		},
		GetStatus: func() string {
			return "Running"
		},
		// Serialization functions
		SerializeConfig: func(config *CustomizedConfig) string {
			return fmt.Sprintf("{\"value\":%d}", config.Value)
		},
		DeserializeConfig: func(configStr string) (*CustomizedConfig, error) {
			var cfg CustomizedConfig
			if err := json.Unmarshal([]byte(configStr), &cfg); err != nil {
				return nil, err
			}
			return &cfg, nil
		},
		SerializeOutput: func(output CustomizedOutput) string {
			return fmt.Sprintf("{\"count\":%d,\"timestamp\":\"%s\"}",
				output.Count,
				output.Timestamp.Format(time.RFC3339))
		},
	}
}
