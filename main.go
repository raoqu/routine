package main

import (
	"flag"
	"log"
)

// No global flags - all state is now maintained in the RoutineScheduler instance

func main() {
	// Parse command line flags
	interactiveFlag := flag.Bool("interactive", true, "Run in interactive mode with UI")
	portFlag := flag.Int("port", 8080, "Port to run the server on")
	flag.Parse()

	// Use the specified port or default to 8080
	port := *portFlag

	log.Println("Starting Routine Manager application...")
	if !*interactiveFlag {
		log.Println("Running in non-interactive mode - UI will be hidden")
	}
	log.Printf("Using port: %d", port)

	// Create a local scheduler instance with a routine instance
	routine := NewCustomizedRoutine()
	scheduler := NewRoutineScheduler[*CustomizedConfig, CustomizedOutput](port, routine, *interactiveFlag)

	// Start some test routines if in non-interactive mode
	if !scheduler.InteractiveMode {
		log.Println("Starting test routines...")
	}

	// Start the server
	scheduler.Serve()
}
