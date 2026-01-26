## sync.Mutex 介绍

`sync.Mutex`是 Go 语言标准库提供的互斥锁，用于保护共享资源。

当多个 goroutine 同时读写同一个共享变量时，如果没有适当的同步机制，程序的行为将变得不可预测。

为了防止多个 goroutine 同时访问共享数据造成数据竞争（race condition），它通过互斥的方式确保同一时间只有一个 goroutine 能够访问临界区代码，从而避免数据竞争。

理解 Mutex 的关键在于理解"临界区"（Critical Section）的概念。临界区是指访问共享资源（如全局变量、共享结构体、文件等）的那段代码。任何时候，我们都必须确保最多只有一个 goroutine 处于临界区中，这就是互斥的核心思想。

Mutex 的工作原理其实类似于一个房间的钥匙：只有拿到钥匙的人才能进入房间（执行临界区代码），其他人只能在门外等待。

## sync.Mutex 功能

在 Go 中，`sync.Mutex` 是一个结构体类型，它提供两个核心方法：`Lock()` 用于获取锁，`Unlock()` 用于释放锁。当一个 goroutine 成功调用 `Lock()` 后，其他尝试获取同一把锁的 goroutine 将会阻塞，直到持有锁的 goroutine 调用 `Unlock()` 释放锁为止。

**核心特性分析：**

- **互斥性**：同一时刻只允许一个 goroutine 持有锁
- **可重入性**：Mutex 是不可重入的，同一 goroutine 重复加锁会导致死锁
- **零值可用**：Mutex 的零值即为未加锁状态，可以直接使用
- **非公平锁**：等待的 goroutine 不保证按 FIFO 顺序获取锁

**基本方法：**

- `Lock()`：获取锁，如果锁已被占用则阻塞等待
- `Unlock()`：释放锁

## 常见使用场景

Mutex 主要用于以下场景：

1. **保护共享变量**：多个 goroutine 读写同一变量时
2. **保护数据结构**：如 map、slice 等非并发安全的结构
3. **临界区保护**：需要原子性执行的代码段
4. **资源访问控制**：文件、网络连接等资源的并发访问

## 使用方法与代码示例

### 基本使用

**第一个例子**：

不使用 mutex 同步代码，有并发操作时没能产生正确的数据，

 mutex/basic/basic0.go

```go
package main

import (
    "fmt"
    "time"
)

var count int

func main() {
    for i := 0; i < 1000; i++ {
        go add()
    }
    time.Sleep(1 * time.Second)
    fmt.Println("count:", count)
}

func add() {
    count++
}

/**

go run ./basic0.go
count: 986
**/
```

使用 mutex 同步代码，产生正确的数据，

basic.go

```go
package main

import (
    "fmt"
    "sync"
    "time"
)

var (
    count int
    mutex sync.Mutex
)

func main() {
    for i := 0; i < 1000; i++ {
        go add()
    }
    time.Sleep(1 * time.Second)
    fmt.Println("count:", count)
}

func add() {
    mutex.Lock()
    count++
    mutex.Unlock()
}

/**

go run ./basic.go
count: 1000
**/
```

**第二个例子**：

结构体嵌入mutex 例子，codes/mutex/basic/basic1.go

```go
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
```

在这个例子中，我们创建了一个 `Counter` 结构体，并在其中嵌入了 `sync.Mutex`。`Inc` 方法和 `Value` 方法都在访问共享的 `value` 字段之前调用 `Lock()`，并在操作完成后通过 `defer` 调用 `Unlock()`

### 保护复杂的数据结构

用来保护复杂数据 map，对 map 的操作进行加锁（mutex）操作。

最后 sync.Mutex 结合 sync.WaitGroup 进行操作。

代码例子 codes/mutex/safemap/safemap.go：

```go
package main

import (
	"fmt"
	"sync"
)

type SafeMap struct {
	mu sync.Mutex
	m map[string]int
}

func NewSafeMap() *SafeMap {
	return &SafeMap{
		m: make(map[string]int),
	}
}

// Set 用于设置键值对
func (s *SafeMap) Set(key string, value int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = value
}

// Get 用于获取健的值
func (s *SafeMap) Get(key string) (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	val, ok :=s.m[key]
	return val, ok
}

// Delete 方法用于删除指定键
func (s *SafeMap) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, key)
}

// Keys 方法返回所有键的切片
func (s *SafeMap) Keys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	keys := make([]string, 0, len(s.m))
	for k := range s.m {
		keys =append(keys, k)
	}
	return keys
}

func main() {
	s := NewSafeMap()

	// 并发写数据，启动 30 个goroutine写
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", id)
			s.Set(key, id*10)
		}(i)
	}

	wg.Wait()

	for _, k:= range s.Keys() {
		if v, ok := s.Get(k); ok {
			fmt.Printf("%s = %d \n", k, v) // 打印所有数据
		}
	}
}
```

上面的例子是将 Mutex 作为结构体的普通字段，对复杂数据结构进行保护。

### 任务调度1：简单例子

写一个复杂点的任务调度例子

mutex/taskschedu/tasksched.go

```go
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
```

### 任务调度2：限制并发任务数量

在并发系统中，有时需要限制同时执行的任务数量。另外一种实现的调度器。

mutex/taskschedu/taskrun.go

```go
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
```

这个例子展示了使用 Mutex 配合 channel 来实现更复杂的并发控制模式。`TaskRunner` 不仅使用 channel 来限制并发数，还使用 Mutex 来追踪当前活跃的任务数量。

## 常见错误和正确实践

### 错误一：复制 Mutex

```go
package main

import (
    "sync"
)

type BadCounter struct {
    value int
    mu    sync.Mutex  // 嵌入式 Mutex
}

// ❌错误：值接收者会复制 Mutex
func (c BadCounter) Increment() {
    c.mu.Lock()
    c.value++
    c.mu.Unlock()
}

// ✅正确：使用指针接收者
func (c *BadCounter) IncrementCorrect() {
    c.mu.Lock()
    c.value++
    c.mu.Unlock()
}

func main() {
    counter := BadCounter{value: 0}
    
    // 注意：这里的调用实际上不会锁住原始的 Mutex
    // 因为值接收者创建了一个副本
    counter.Increment()  // 数据竞争！
    
    // 正确的方式
    counter.IncrementCorrect()  // 正常工作
}
```

这是一个非常隐蔽但危险的错误。当使用值接收者调用方法时，Mutex 会被复制，而复制后的 Mutex 是一个全新的锁，与原始锁完全无关。最佳实践是始终对包含 Mutex 的结构体使用指针接收者，或者确保 Mutex 不会被复制。

Go 1.9 之后，使用`go vet` 工具可以检测到这种问题。

### 错误二：忘记释放锁

```go
func (c *Counter) ForgotUnlock() {
    c.mu.Lock()
    if someCondition() {
        return  // ❌错误：在这里返回，锁永远不会被释放
    }
    c.value++
    c.mu.Unlock()  // 这行代码可能永远不会执行
}

// ✅正确：使用 defer
func (c *Counter) RememberUnlock() {
    c.mu.Lock()
    defer c.mu.Unlock()
    if someCondition() {
        return  // defer 会确保锁被释放
    }
    c.value++
}
```

忘记释放锁是最常见的错误之一，会导致所有等待该锁的 goroutine 永久阻塞。使用 `defer` 可以有效地避免这个问题，它确保在函数返回前锁一定会被释放。

### 错误三：Mutex 不支重复加锁

递归加锁

```go
func (c *Counter) Recursive() {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // ❌ 错误：尝试在已持有锁的情况下再次加锁会导致死锁
    c.mu.Lock()  // 这里会永久阻塞，因为当前 goroutine 已经持有锁
    c.mu.Unlock()
}
```

重复加锁

```go
// ❌错误：同一个 goroutine 重复加锁
func (c *Counter) BadMethod() {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // 这里会死锁！
    c.AnotherMethod()  // AnotherMethod 内部也会调用 Lock()
}

func (c *Counter) AnotherMethod() {
    c.mu.Lock()  // 死锁！
    defer c.mu.Unlock()
    // ...
}
```

与某些其他语言的 Mutex 实现不同，Go 的 `sync.Mutex` 不支持重复加锁（也称为可重入锁）。如果同一个 goroutine 尝试两次获取同一把锁，第二次调用会永久阻塞。

这是有意为之的设计，因为它简化了 Mutex 的实现并帮助开发者发现潜在的设计问题。

### 错误四：持有锁时间过长

```go
// ❌错误：在锁内执行耗时操作
func (s *Service) BadProcess() {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // ❌错误：耗时的网络请求在锁内
    data := fetchFromNetwork()  // 可能需要几秒钟
    s.cache = data
}

// ✅正确：缩小锁范围
func (s *Service) GoodProcess() {
    data := fetchFromNetwork()  // 在锁外执行
    
    s.mu.Lock()
    s.cache = data
    s.mu.Unlock()
}
```

不要持有锁时间过长，尽量减少锁的力度。

### 正确实践一：使用 defer 释放锁

使用 defer 释放锁

```go
func (c *Counter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock()  // 使用defer确保函数返回时一定会解锁
    c.value++
}
```

### 正确实践二：锁的粒度最小化

```go
// 好的做法：缩小锁的范围
func (s *Service) ProcessData(data string) error {
    // 耗时的操作放在锁外面
    processed := heavyProcessing(data)
    
    // 只在必要时加锁
    s.mu.Lock()
    s.cache[data] = processed
    s.mu.Unlock()
    
    return nil
}
```

### 正确实践三：锁应该保护数据而非代码

```go
// 好的设计：锁和数据封装在一起
type SafeCounter struct {
    mu    sync.Mutex
    count int  // 锁保护这个字段
}

func (c *SafeCounter) Inc() {
    c.mu.Lock()
    c.count++
    c.mu.Unlock()
}
```

### 正确实践四：用RWMutex优化读多写少场景

```go
type ReadCache struct {
    mu   sync.RWMutex  // 读写锁
    data map[string]string
}

func (c *ReadCache) Get(key string) (string, bool) {
    c.mu.RLock()  // 读锁，允许多个goroutine同时读
    defer c.mu.RUnlock()
    val, ok := c.data[key]
    return val, ok
}

func (c *ReadCache) Set(key, value string) {
    c.mu.Lock()  // 写锁，独占访问
    defer c.mu.Unlock()
    c.data[key] = value
}
```



还有哪些错误使用 mutex 和 正确使用 mutex 的实践？欢迎大家给出代码或评论。
