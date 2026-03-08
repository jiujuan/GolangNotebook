Go 语言编程中常用的并发场景和并发模式，代码例子说明。

## Worker Pool（工作池模式）

需要限制并发数量，处理大量任务时避免创建过多 goroutine。

启动固定数量的 goroutine，让它们竞争读取同一个任务 Channel。

这个工作池包含了任务分发、并发处理、结果收集，以及优雅关闭

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

## Pipeline（管道模式）

数据需要经过多个阶段处理，每个阶段独立完成特定任务。

```go
package main

import (
	"fmt"
)

// Pipeline 模式：生成器 -> 处理器 -> 输出
func pipelinePattern() {
	// 阶段1：生成数字
	generator := func(nums ...int) <-chan int {
		out := make(chan int)
		go func() {
			defer close(out)
			for _, n := range nums {
				out <- n
			}
		}()
		return out
	}

	// 阶段2：平方
	square := func(in <-chan int) <-chan int {
		out := make(chan int)
		go func() {
			defer close(out)
			for n := range in {
				out <- n * n
			}
		}()
		return out
	}

	// 阶段3：过滤偶数
	filterEven := func(in <-chan int) <-chan int {
		out := make(chan int)
		go func() {
			defer close(out)
			for n := range in {
				if n%2 == 0 {
					out <- n
				}
			}
		}()
		return out
	}

	// 构建 pipeline
	nums := generator(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	squared := square(nums)
	filtered := filterEven(squared)

	// 消费结果
	for result := range filtered {
		fmt.Printf("Result: %d\n", result)
	}
}
```



