package workerpool

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultJobBufferSize = 100
)

// Job is the type of the job that the worker pool can handle.
type Job func()

// WorkerPool is the worker pool that can handle the job.
type WorkerPool struct {
	// size of worker pool, default is runtime.NumCPU().
	size int

	// jobsChan is used to send jobs to the pool
	// it defaults is 100.
	jobsChan chan Job

	// stopOnce is used to ensure the pool is stopped only oncel.
	stopOnce sync.Once

	// stopFlag is used to check if the pool is stopped.
	stopFlag atomic.Bool

	poolCtx    context.Context
	poolCancel context.CancelFunc
	wg         *sync.WaitGroup
}

// Option represents the optional function.
type Option func(*WorkerPool)

// NewWorkerPool creates a new worker pool.
// size is the size of the worker pool, default is runtime.NumCPU().
// options is the customized options.
func NewWorkerPool(size int, options ...Option) *WorkerPool {
	pool := &WorkerPool{
		jobsChan: make(chan Job, defaultJobBufferSize),
		wg:       &sync.WaitGroup{},
		size:     size,
	}

	pool.poolCtx, pool.poolCancel = context.WithCancel(context.Background())

	for _, option := range options {
		option(pool)
	}

	if pool.size <= 0 {
		pool.size = runtime.NumCPU()
	}

	for i := 1; i <= pool.size; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}

	return pool
}

// WithJobBufferSize is an option to set the buffer size of the jobs channel
func WithJobBufferSize(size int) Option {
	return func(w *WorkerPool) {
		if size < 0 {
			size = defaultJobBufferSize
		}
		w.jobsChan = make(chan Job, size)
	}
}

// worker is a worker that listens for jobs
func (w *WorkerPool) worker() {
	defer w.wg.Done()

	for {
		select {
		case job, ok := <-w.jobsChan:
			if !ok || w.poolCtx.Err() != nil {
				return
			}

			job()
		case <-w.poolCtx.Done():
			return
		}
	}
}

// AddTask is used to add a job to the pool
// it returns an error if the pool is stopped
func (w *WorkerPool) AddTask(job Job) error {
	if w.stopFlag.Load() {
		return ErrPoolClosed
	}

	w.jobsChan <- job

	return nil
}

// ShutdownWithTimeout shuts down the worker pool with a timeout
func (w *WorkerPool) ShutdownWithTimeout(timeout time.Duration) error {
	var err error
	w.stopOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		w.stopFlag.Store(true)
		w.poolCancel()

		done := make(chan struct{})
		go func() {
			w.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			close(w.jobsChan)
		case <-ctx.Done():
			close(w.jobsChan)
			err = ctx.Err()
		}
	})

	return err
}

// ForceShutdown shuts down the worker pool immediately,
// it will not wait for the jobs to complete,
// but it will lead to the worker keep running and memory leak
func (w *WorkerPool) ForceShutdown() {
	w.stopOnce.Do(func() {
		w.stopFlag.Store(true)
		w.poolCancel()
		close(w.jobsChan)
	})
}

// WaitAndShutdown waits for all queued jobs to complete before shutting down
func (w *WorkerPool) WaitAndShutdown() {
	w.stopOnce.Do(func() {
		w.stopFlag.Store(true)

		remainingJobs := len(w.jobsChan)
		if remainingJobs > 0 {
			originalJobs := make([]Job, 0, remainingJobs)
			draining := true
			for draining {
				select {
				case job, ok := <-w.jobsChan:
					if ok {
						originalJobs = append(originalJobs, job)
					}
				default:
					draining = false
				}
			}
			tasksDrained := make(chan struct{}, len(originalJobs))

			for _, job := range originalJobs {
				wrappedJob := func(originalJob Job) Job {
					return func() {
						originalJob()
						tasksDrained <- struct{}{}
					}
				}(job)
				w.jobsChan <- wrappedJob
			}

			for i := 0; i < len(originalJobs); i++ {
				<-tasksDrained
			}
		}

		w.poolCancel()
		w.wg.Wait()
		close(w.jobsChan)
	})
}
