package main

import (
	"fmt"
	"sync"
	"time"
	worker_pool "worker-pool"
)

func main() {
	workerPool := worker_pool.NewWorkerPool(10,worker_pool.WithInputChan(1000), worker_pool.WithJobChan(5))

	wg := sync.WaitGroup{}
	for i:=0;i<100;i++ {
		wg.Add(1)
		workerPool.AddTask(func(params []interface{}) interface{} {
			defer wg.Done()
			fmt.Printf("Task Working [%v] \n", params[0])
			time.Sleep(3 * time.Second)
			return nil
		}, i)
	}
	fmt.Println("do shutdown")
	workerPool.Shutdown()
	wg.Wait()
	fmt.Println("Done")
}
