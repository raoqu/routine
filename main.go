package main

import (
	"log"
)

func main() {
	// Initialize the scheduler with port 8080
	log.Println("Starting Routine Manager application...")
	scheduler = NewRoutineScheduler(8080)

	// This should block until the server exits
	scheduler.Serve()

	// This line should never be reached unless the server exits
	log.Println("Server has stopped.")
}
