package main

import (
	"flag"
	"log"
)

// Global flag to indicate if we're in test mode
// This is used by both main.go and scheduler.go
// IsTestMode indicates whether the application is running in test mode
var IsTestMode bool

func main() {
	// Parse command line flags
	testFlag := flag.Bool("test", false, "Run in test mode")
	portFlag := flag.Int("port", 8080, "Port to run the server on")
	flag.Parse()

	// Set the global test mode flag
	IsTestMode = *testFlag

	// Use the specified port or default to 8080
	port := *portFlag

	log.Println("Starting Routine Manager application...")
	if IsTestMode {
		log.Println("Running in test mode - UI will be hidden")
	}
	log.Printf("Using port: %d", port)

	// Create a local scheduler instance with a routine instance
	routine := NewCustomizedRoutine()
	scheduler := NewRoutineScheduler[*CustomizedConfig, CustomizedOutput](port, routine)

	// Start some test routines if in test mode
	if IsTestMode {
		log.Println("Starting test routines...")
	}

	// Start the server
	scheduler.Serve()
}
