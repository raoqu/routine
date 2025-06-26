package main

import (
	"log"
)

func main() {
	// Initialize the scheduler with port 8080 and CustomizedRoutine creator
	log.Println("Starting Routine Manager application...")
	scheduler = NewRoutineScheduler[CustomizedConfig, CustomizedOutput](8080, NewCustomizedRoutine)

	// This should block until the server exits
	scheduler.Serve()

	// This line should never be reached unless the server exits
	log.Println("Server has stopped.")
}
