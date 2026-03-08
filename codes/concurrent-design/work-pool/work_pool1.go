package main

import (
	"fmt"
	"sync"
	"time"
)

// 定义任务结构
type Job struct {
	ID int
}

type Result struct {
	JobID  int
	Worker int
	Value  int
}

func worker(id int, jobs <-chan Job, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done() // 完成后通知 WaitGroup

	for job := range jobs { // 阻塞直到有任务进入 Channel 或 Channel 关闭
		fmt.Printf("Worker %d 开始处理任务 %d\n", id, job.ID)
		// 模拟耗时操作
		time.Sleep(time.Millisecond * 500)

		// 发送结果
		results <- Result{JobID: job.ID, Worker: id, Value: job.ID * 2}
	}

	fmt.Printf("Worker %d 退出\n", id)
}

func main() {
	const numJobs = 10   // 总任务数
	const numWorkers = 3 // 并发限制（池大小）

	jobs := make(chan Job, numJobs)
	results := make(chan Result, numJobs)

	var wg sync.WaitGroup

	// 1.启动workers
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go worker(w, jobs, results, &wg)
	}

	// 2.发送任务
	for j:=1; j<=numJobs;j++{
		jobs<-Job{ID:j}
	}
	close(jobs) // 发送完毕后必须关闭，worker 里的 range 才会结束


	// 3.等待所有 Worker 结束
	// 注意：在另一个协程中等待，避免阻塞主流程收集结果
	go func() {
		wg.Wait()
		close(results) // 所有任务完成后关闭结果通道
	}

	// 4.收集结果
	for res := range results {
		fmt.Printf("结果: 任务 %d 由 Worker %d 完成，结果为 %d\n", 
            res.JobID, res.Worker, res.Value)
	}

}
