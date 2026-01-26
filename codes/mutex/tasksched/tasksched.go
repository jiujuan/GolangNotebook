package main

import (
	"fmt"
	"sync"
	"time"
)

type TaskScheduler struct {
	mu sync.Mutex
	tasks []Task
	active bool
}

type Task struct {
	ID int
	Name string
	Execute func() // 执行任务的函数
	Priority int
}

func NewTaskScheduler() *TaskScheduler {
	return &TaskScheduler{
		tasks: make([]Task, 0),
	}
}

// AddTask 添加任务
func (ts *TaskScheduler) AddTask(task Task) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.tasks = append(ts.tasks, task)
}

// 获取下一个任务
func (ts *TaskScheduler) GetNextTask() (Task, bool) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if len(ts.tasks) == 0 {
		return Task{}, false
	}

	task := ts.tasks[0]
	ts.tasks = ts.tasks[1:]
	return task, true
}

func (ts *TaskScheduler) Start(workers int) {
	ts.mu.Lock()
	ts.active = true
	ts.mu.Unlock()

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				task, ok := ts.GetNextTask()
				if !ok { // 获取任务失败
					time.Sleep(100 * time.Millisecond)
					ts.mu.Lock()
					active := ts.active // 查看是否有停止任务
					ts.mu.Unlock()
					if !active { // 如果有停止执行的任务， 跳出当前循环，不执行下面的任务
						break // 跳出当前循环，也就是不执行此次的任务
					}
					continue
				}
				
				fmt.Printf("工作协程worker ID: %d Task任务Name: %s\n", id, task.Name)
				task.Execute() // 执行任务
			}

		}(i)

	}

	wg.Wait()
}

// 停止执行任务
func (ts *TaskScheduler) Stop() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.active = false
}

func main() {
	scheduler := NewTaskScheduler()

	// 添加任务，准备 10 个任务
	for i :=0; i < 10; i++ {
		taskID := i
		Name := fmt.Sprintf("Task-%d", taskID)
		scheduler.AddTask(Task{
			ID: taskID,
			Name: Name,
			Execute: func() {
				time.Sleep(200 * time.Millisecond)
				fmt.Printf("Task任务ID: %d 任务name: %s\n", taskID, Name)
			},
		})
	}


	go func() {
        time.Sleep(3 * time.Second)
        scheduler.Stop()
    }()

    // 启动 3 个工作协程来执行任务
    fmt.Println("启动 3 个工作协程来执行任务")
    scheduler.Start(3)
}