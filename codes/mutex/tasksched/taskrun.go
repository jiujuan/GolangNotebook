package main

import (
	"fmt"
	"sync"
	"time"
)

// Semaphore 实现一个信号量来限制并发数
type Semaphore struct {
	mu sync.Mutex
	slots chan struct{}
}

// TaskRunner 限制并发任务数的执行器
type TaskRunner struct {
    mu          sync.Mutex
    semaphore   *Semaphore
    activeCount int
    maxActive   int
}

// NewSemaphore 创建一个带有指定并发限制的信号量
func NewSemaphore(maxConcurrency int) *Semaphore {
	return &Semaphore{
        slots: make(chan struct{}, maxConcurrency),
    }
}

// Acquire 获取一个信号量槽位，如果槽位已满则阻塞
func (s *Semaphore) Acquire() {
	s.slots <- struct{}{}
}

// Release 释放一个信号量槽位
func (s *Semaphore) Release() {
	select {
	case <- s.slots:
	default:
		// 槽位为空时不做任何操作
	}
}

// NewTaskRunner 创建一个新的任务执行器
func NewTaskRunner(maxActive int) *TaskRunner {
    return &TaskRunner{
        semaphore: NewSemaphore(maxActive),
        maxActive: maxActive,
    }
}

// Run 执行一个任务，限制同时运行的任务数量
func (r *TaskRunner) Run(task func()) {
    r.semaphore.Acquire()
    
    r.mu.Lock()
    r.activeCount++
    current := r.activeCount
    r.mu.Unlock()
    
    fmt.Printf("Task 开始, active tasks: %d/%d\n", current, r.maxActive)
    
    // 执行任务
    task()
    
    // 任务完成
    r.mu.Lock()
    r.activeCount--
    r.mu.Unlock()
    
    r.semaphore.Release()
    fmt.Printf("Task 完成, active tasks 释放减少\n")
}

func main() {
	// 最多运行4个任务
	runner :=NewTaskRunner(4)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		taskID := i
		go func() {
			defer wg.Done()
			runner.Run(func() {
				fmt.Printf("执行Processing task %d\n", taskID)
                time.Sleep(500 * time.Millisecond)
			})
		}()
	}

	wg.Wait()
}