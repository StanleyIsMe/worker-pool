package worker_pool


import (
	"context"
	"sync"
)

const (
	PoolSize     = 8
	inputChannel = 100
	jobChannel   = 100
)

type TaskMethod func(params []interface{}) interface{}


type TaskParam struct {
	TaskMethod TaskMethod
	TaskParam  []interface{}
}

type WorkerPool struct {
	inputChan chan *TaskParam
	jobsChan  chan *TaskParam
	stopOnce sync.Once
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup
}



func NewWorkerPool(maxWorkers int, options ...func(*WorkerPool)) *WorkerPool {
	pool := &WorkerPool{
		inputChan:  make(chan *TaskParam, inputChannel),
		jobsChan: make(chan *TaskParam, jobChannel),
		wg:        &sync.WaitGroup{},
	}
	pool.ctx, pool.cancel = context.WithCancel(context.Background())

	for _, option := range options {
		option(pool)
	}

	if maxWorkers <= 0 {
		maxWorkers = PoolSize
	}

	for i := 0; i < maxWorkers; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}

	go pool.listen()
	return pool
}

func WithJobChan(jobNum int) func(*WorkerPool) {
	return func(w *WorkerPool) {
		if jobNum < 0 {
			jobNum =  jobChannel
		}
		w.jobsChan = make(chan *TaskParam, jobNum)
	}
}

func WithInputChan(inputNum int) func(*WorkerPool) {
	return func(w *WorkerPool) {
		if inputNum < 0 {
			inputNum =  inputChannel
		}
		w.inputChan = make(chan *TaskParam, inputNum)
	}
}


func (w *WorkerPool) listen() {
	defer close(w.jobsChan)
	for {
		select {
		case job, ok := <-w.inputChan:
			if w.ctx.Err() != nil && !ok {
				return
			}
			w.jobsChan <- job
		}
	}
}

func (w *WorkerPool) worker() {
	defer w.wg.Done()
	for {
		select {
		case job, ok := <-w.jobsChan:
			if w.ctx.Err() != nil && !ok {
				return
			}
			job.TaskMethod(job.TaskParam)
		case <-w.ctx.Done():
			if len(w.jobsChan) > 0 {
				continue
			}
			return
		}
	}
}

func (w *WorkerPool) AddTask(task TaskMethod, params ...interface{}) {
	w.inputChan <- &TaskParam{task, params}
}

func (w *WorkerPool) Shutdown() {
	w.stopOnce.Do(func() {
		w.cancel()
		w.wg.Wait()
		close(w.inputChan)
	})
}