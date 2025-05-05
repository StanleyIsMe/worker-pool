package workerpool_test

import (
	"fmt"
	"workerpool"
)

// Simple example
func ExampleWorkerPool() {
	pool := workerpool.NewWorkerPool(10)
	pool.AddTask(func() {
		fmt.Println("Hello, World!")
	})
	pool.WaitAndShutdown()

	// Output:
	// Hello, World!
}
