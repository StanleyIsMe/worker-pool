package workerpool

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerPool(t *testing.T) {
	t.Run("basic functionality", func(t *testing.T) {
		pool := NewWorkerPool(2)
		var vcounter atomic.Int32

		for i := 0; i < 5; i++ {
			err := pool.AddTask(func() {
				vcounter.Add(1)
			})
			if err != nil {
				t.Errorf("Failed to add task: %v", err)
			}
		}

		// Wait for tasks to complete
		time.Sleep(100 * time.Millisecond)
		if vcounter.Load() != 5 {
			t.Errorf("Expected counter to be 5, got %d", vcounter.Load())
		}

		pool.WaitAndShutdown()
	})

	t.Run("with custom buffer size", func(t *testing.T) {
		pool := NewWorkerPool(10, WithJobBufferSize(10000))
		var counter atomic.Int32

		for i := 0; i < 10000; i++ {
			err := pool.AddTask(func() {
				time.Sleep(time.Millisecond)
				counter.Add(1)
			})
			if err != nil {
				t.Errorf("Failed to add task: %v", err)
			}
		}

		pool.WaitAndShutdown()
		if counter.Load() != 10000 {
			t.Errorf("Expected counter to be 10000, got %d", counter.Load())
		}
	})

	t.Run("invalid buffer size", func(t *testing.T) {
		pool := NewWorkerPool(2, WithJobBufferSize(-1))
		if cap(pool.jobsChan) != defaultJobBufferSize {
			t.Errorf("Expected buffer size to be %d, got %d", defaultJobBufferSize, cap(pool.jobsChan))
		}
		pool.ForceShutdown()
	})

	t.Run("zero workers", func(t *testing.T) {
		pool := NewWorkerPool(0)
		if pool.size == 0 {
			t.Errorf("Expected size must be greater than 0, got %d", pool.size)
		}
		pool.ForceShutdown()
	})

	t.Run("shutdown with timeout", func(t *testing.T) {
		pool := NewWorkerPool(2)

		// Add a long-running task
		err := pool.AddTask(func() {
			time.Sleep(10 * time.Second)
		})
		if err != nil {
			t.Errorf("Failed to add task: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
		err = pool.ShutdownWithTimeout(10 * time.Millisecond)
		if err == nil {
			t.Error("Expected timeout error, got nil")
		}
	})

	t.Run("job finished before shutdown", func(t *testing.T) {
		pool := NewWorkerPool(2, WithJobBufferSize(10000))

		// Add a long-running task
		err := pool.AddTask(func() {
			time.Sleep(time.Millisecond)
		})
		if err != nil {
			t.Errorf("Failed to add task: %v", err)
		}

		time.Sleep(10 * time.Millisecond)
		if err := pool.ShutdownWithTimeout(100 * time.Millisecond); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("add task after shutdown", func(t *testing.T) {
		pool := NewWorkerPool(2)
		pool.ForceShutdown()

		err := pool.AddTask(func() {})
		if err != ErrPoolClosed {
			t.Errorf("Expected ErrPoolClosed, got %v", err)
		}
	})

	t.Run("multiple shutdowns", func(t *testing.T) {
		pool := NewWorkerPool(2)
		pool.ForceShutdown()
		pool.ForceShutdown()
		pool.WaitAndShutdown()
		err := pool.ShutdownWithTimeout(time.Second)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}
