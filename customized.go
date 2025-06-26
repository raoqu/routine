package main

import (
	"fmt"
	"time"
)

type CustomizedConfig struct {
	Value int
}

type CustomizedRoutine struct {
}

func (r *CustomizedRoutine) Job(config *CustomizedConfig) int {
	return config.Value
}

func (r *CustomizedRoutine) GenIdentity(config *CustomizedConfig) string {
	// generate id with nano timestamp and random number
	return fmt.Sprintf("CR-%d-%d", time.Now().UnixNano(), config.Value)
}

func (r *CustomizedRoutine) GetStatus() string {
	return "Running"
}
