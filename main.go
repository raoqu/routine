package main

func main() {
	// Initialize the scheduler with port 8080
	scheduler = NewRoutineScheduler(8080)

	scheduler.Serve()
}
