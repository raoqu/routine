package main

import (
	"fmt"
	"log"
	"time"
)

// CustomizedConfig holds the configuration for a CustomizedRoutine
type CustomizedConfig struct {
	Value int
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
func NewCustomizedRoutine() *Routine[CustomizedConfig, CustomizedOutput] {
	return &Routine[CustomizedConfig, CustomizedOutput]{
		ID: "customized",
		Job: func(ctrl *RoutineControl[CustomizedConfig, CustomizedOutput]) CustomizedOutput {
			config := ctrl.Config.Load().(CustomizedConfig)
			prevOutput := ctrl.Output.Load().(CustomizedOutput)

			// Create new output with incremented count
			newOutput := CustomizedOutput{
				Count:     prevOutput.Count + config.Value,
				Timestamp: time.Now(),
			}

			log.Printf("CustomizedRoutine running: count=%d config=%d", newOutput.Count, config.Value)
			return newOutput
		},
		GenIdentity: func(config CustomizedConfig) string {
			// Generate ID with nano timestamp and config value
			return fmt.Sprintf("CR-%d-%d", time.Now().UnixNano(), config.Value)
		},
		GetStatus: func() string {
			return "Running"
		},
		// Serialization functions
		SerializeConfig: func(config CustomizedConfig) string {
			return fmt.Sprintf("{\"value\":%d}", config.Value)
		},
		DeserializeConfig: func(configStr string) CustomizedConfig {
			var value int
			// Simple parsing, in production would use proper JSON parsing
			fmt.Sscanf(configStr, "{\"value\":%d}", &value)
			return CustomizedConfig{Value: value}
		},
		SerializeOutput: func(output CustomizedOutput) string {
			return fmt.Sprintf("{\"count\":%d,\"timestamp\":\"%s\"}",
				output.Count,
				output.Timestamp.Format(time.RFC3339))
		},
		DeserializeOutput: func(outputStr string) CustomizedOutput {
			var count int
			var timeStr string
			// Simple parsing, in production would use proper JSON parsing
			fmt.Sscanf(outputStr, "{\"count\":%d,\"timestamp\":\"%s\"}", &count, &timeStr)
			t, _ := time.Parse(time.RFC3339, timeStr)
			return CustomizedOutput{Count: count, Timestamp: t}
		},
	}
}
