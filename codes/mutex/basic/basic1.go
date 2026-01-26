package main

import (

	"fmt"
	"sync"
)

type Counter struct {
	mu sync.Mutex
	num int
}

func (c *Counter) Inc() {
	c.mu.Lock() // Lock 方法获取锁，如果锁已被其他 goroutine 持有，则阻塞等待
	defer c.mu.Unlock() // 使用 defer 确保即使发生 panic，锁也能被正确释放

	c.num++ // 临界区：只有持有锁的 goroutine 可以安全地访问 c.num
}

// 用于获取当前计数器值
func (c *Counter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.num
}

func main() {
	var counter Counter

	// 启动 1000 个 goroutine 并发地增加计数器
	var wg sync.WaitGroup
	for i :=0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter.Inc()
		}()
	}

	wg.Wait()

	fmt.Printf("num Value: %d \n", counter.Value())
}