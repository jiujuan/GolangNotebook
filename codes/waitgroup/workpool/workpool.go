package main

import (
	"fmt"
	"sync"
	"time"
)

// Job 代表需要处理的任务
type Job struct {
	ID   int
	Data string
}

// WorkerPool 管理一组 worker goroutine
type WorkerPool struct {
	workerCount int
	jobs        chan Job
	wg          sync.WaitGroup
}

// NewWorkerPool 创建一个新的 worker 池
func NewWorkerPool(workerCount int) *WorkerPool {
	return &WorkerPool{
		workerCount: workerCount,
		jobs:        make(chan Job, 100), // 缓冲队列
	}
}

// Start 启动所有 worker
func (pool *WorkerPool) Start() {
	for i := 0; i < pool.workerCount; i++ {
		pool.wg.Add(1)
		go pool.worker(i)
	}
}

// Submit 提交一个任务到队列
func (pool *WorkerPool) Submit(job Job) {
	// 在实际应用中可能需要处理队列满的情况
	select {
	case pool.jobs <- job:
	default:
		fmt.Printf("任务队列已满，任务 %d 被拒绝\n", job.ID)
	}
}

// worker 是实际的 worker goroutine
func (pool *WorkerPool) worker(id int) {
	defer pool.wg.Done()
	for job := range pool.jobs {
		fmt.Printf("Worker %d 开始处理任务 %d\n", id, job.ID)
		time.Sleep(100 * time.Millisecond) // 模拟任务处理
		fmt.Printf("Worker %d 完成处理任务 %d\n", id, job.ID)
	}
}

// Stop 停止 worker 池，等待所有任务完成
func (pool *WorkerPool) Stop() {
	close(pool.jobs) // 关闭通道会触发所有 worker 退出循环
    pool.wg.Wait()       // 等待所有 worker 完成
	fmt.Println("所有worker已停止")
}

func main() {
	pool := NewWorkerPool(3)
	pool.Start()

	// 提交6个任务
	for i := 1; i <= 6; i++ {
		pool.Submit(Job{ID: i, Data: fmt.Sprintf("数据-%d", i)})
	}

	// 给 worker 一些时间处理任务
	time.Sleep(time.Second)
	pool.Stop()
	fmt.Println("程序正常退出")
}