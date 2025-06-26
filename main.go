package main

import (
	"flag"
	"log"
)

// Global flag to indicate if we're in test mode
// This is used by both main.go and scheduler.go
var isTestMode bool

func main() {
	// Parse command line flags
	testFlag := flag.Bool("test", false, "Run in test mode")
	portFlag := flag.Int("port", 8080, "Port to run the server on")
	flag.Parse()
	
	// Set the global test mode flag
	isTestMode = *testFlag
	
	// Use the specified port or default to 8080
	port := *portFlag
	
	// Initialize the scheduler with the specified port and CustomizedRoutine creator
	log.Println("Starting Routine Manager application...")
	if isTestMode {
		log.Println("Running in test mode - UI will be hidden")
	}
	log.Printf("Using port: %d", port)
	scheduler = NewRoutineScheduler[CustomizedConfig, CustomizedOutput](port, NewCustomizedRoutine)
	
	// Start some test routines if in test mode
	if isTestMode {
		log.Println("Starting test routines...")
		// Start a couple of test routines with different configs
		scheduler.startRoutineWithConfig("test-routine-1", `{"value":1}`)
		scheduler.startRoutineWithConfig("test-routine-2", `{"value":5}`)
		log.Println("Test routines started")
	}

	// This should block until the server exits
	scheduler.Serve()

	// This line should never be reached unless the server exits
	log.Println("Server has stopped.")
}
