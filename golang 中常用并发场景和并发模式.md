Go 语言编程中常用的并发场景和并发模式，代码例子说明。

## Worker Pool（工作池模式）

需要限制并发数量，处理大量任务时避免创建过多 goroutine

```go
package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// 任务结构
type Job struct {
	ID      int
	Data    string
	RetryAt time.Time
}

// 结果结构
type Result struct {
	JobID int
	Value int
	Err   error
}

// Worker Pool 实现
func workerPool() {
	const numWorkers = 3
	const numJobs = 10

	jobs := make(chan Job, numJobs)
	results := make(chan Result, numJobs)

	// 启动 workers
	var wg sync.WaitGroup
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go worker(w, jobs, results, &wg)
	}

	// 发送任务
	go func() {
		for j := 1; j <= numJobs; j++ {
			jobs <- Job{ID: j, Data: fmt.Sprintf("task-%d", j)}
		}
		close(jobs)
	}()

	// 等待所有 workers 完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	for result := range results {
		fmt.Printf("Job %d completed with value: %d\n", result.JobID, result.Value)
	}
}

func worker(id int, jobs <-chan Job, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		fmt.Printf("Worker %d started job %d\n", id, job.ID)
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
		results <- Result{JobID: job.ID, Value: job.ID * 2}
		fmt.Printf("Worker %d finished job %d\n", id, job.ID)
	}
}
```

