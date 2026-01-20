## sync.WaitGroup介绍

`sync.WaitGroup` 是 Go 标准库中用于等待一组 goroutine 完成执行的同步原语，用于协调多个 goroutine 的执行顺序，它提供了一种简单而有效的方式来协调多个并发任务的完成。

在实际开发中，我们启动了多个 goroutine 去并行执行任务，而主 goroutine 需要等待所有子任务完成后才能进行下一步操作或者退出程序，WaitGroup 正是解决这类问题的最佳选择。

> sync.WaitGroup  源码简析：[深入理解Go语言(08)：sync.WaitGroup源码分析](https://www.cnblogs.com/jiujuan/p/16735012.html)



**核心功能如下：**

- 等待一组 goroutine 执行完成
- 阻塞主 goroutine 直到所有子 goroutine 完成
- 线程安全的计数器机制

**主要方法如下：**

- `Add()`: 增加等待的 goroutine 数量。内部实现用来增加或减少内部计数器的值，这个计数器表示还有多少个 goroutine 需要完成。
- `Done()`: 标记一个 goroutine 已完成（相当于 `Add(-1)`）。在每个需要等待的 goroutine 任务结束时调用 Done 方法，告诉WaitGroup 这个任务已经完成。一般用 defer 语句配合 Done 方法来使用。
- `Wait()`: Wait 方法会阻塞调用它的 goroutine，直到 WaitGroup 的计数器变为零。

## 基本使用示例

### 基本的用法

一个简单使用 waitgroup 例子 ，codes/waitgroup/basic/main.go：

```go
package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	var wg sync.WaitGroup

	// 模拟启动 3 个并发任务
	taskCount := 3 
	wg.Add(taskCount)

	for i:=1; i<=taskCount; i++ {
		taskID := i
		go func(){
			defer wg.Done() // 确保函数结束时计数器减一

			fmt.Printf("任务 %d 开始执行 \n", taskID)
			time.Sleep(time.Second * 1) // 模拟耗时操作
			fmt.Printf("任务 %d 执行完成 end\n", taskID)
		}()
	}

	// 等待所有任务完成
	wg.Wait()

	fmt.Println("所有任务已完成，程序即将退出")
}
```

程序说明：

> 首先创建一个 WaitGroup 变量 wg，然后在启动每个 goroutine 之前调用 Add 方法增加计数器，在 goroutine 内部（go func()）使用 defer 调用 Done 方法确保任务结束时计数器减一，最后在需要等待的地方调用 Wait 方法阻塞等待所有任务完成。
>
> 运行这段代码，你会看到三个任务几乎同时开始执行，然后等待一秒后几乎同时完成，最后主程序打印完，程序退出。

用命令 go run 运行程序：

```bash
gogo ❯❯ go run ./main.go
任务 3 开始执行
任务 2 开始执行
任务 1 开始执行
任务 1 执行完成 end
任务 2 执行完成 end
任务 3 执行完成 end
所有任务已完成，程序即将退出
```

### 注意：计数器为负会导致 panic

如果使用不当，可能会导致 go 运行时 panic。

最常见的情况是计数器变为负数，这通常发生在以下几种场景：

- 忘记调用 Add 方法就直接调用 Done 方法，
- 或者在多个 goroutine 中重复调用 Done 方法

以下是会产生 panic 的代码：

```go
package main

import (
	"fmt"
	"sync"
)

func main() {
	var wg sync.WaitGroup

	// 错误：没有先调用 Add，直接调用 Done
	// wg.Done()
	// fmt.Println("这里不会执行，因为上一行会panic")

	// 错误：调用 Done 的次数多于 Add
	wg.Add(1)
	wg.Done()
	wg.Done() // 计数器变为 -1，触发 panic
	fmt.Println("这里不会执行2，因为上一行会panic")
}
```

运行 go run main.go

> go run ./main.go
> panic: sync: negative WaitGroup counter

当运行这段代码时，Go 运行时会检测到 WaitGroup 的计数器变为负数，并抛出类似"sync: negative WaitGroup counter"的 panic 信息。

这个错误信息非常明确地指出了问题所在：在调用 Done 方法之前，Add 方法的调用次数不足以支撑所进行的 Done 调用次数。避免这个问题的关键在于准确控制 Add 和 Done 的调用配对，确保每调用一次 Add，就调用一次 Done。

## 常用使用场景

### 场景1：数据并发处理

在数据处理场景中，我们经常需要将一个大任务拆分成多个小任务并行处理，然后汇总结果。WaitGroup 是实现这种分散-收集模式的理想工具。以下示例展示了如何并行获取多个数据源的数据并聚合结果：

```go
package main

import (
	"fmt"
	"sync"
)

// Result 用于存储各个任务的执行结果
type Result struct {
	Source string
	Data   string
}

func main() {
	var wg sync.WaitGroup
	results := make(chan Result, 3) // 使用缓冲通道存储结果

	// 数据源列表
	sources := []string{"用户数据库", "Redis缓存", "外部API"}

	// 为每个数据源启动一个 goroutine
	for _, source := range sources {
		wg.Add(1)
		go func(src string) {
			defer wg.Done()
			// 模拟从不同数据源获取数据
			data := fetchFromSource(src)
			results <- Result{Source: src, Data: data}
		}(source)
	}

	// 启动一个 goroutine 等待所有任务完成后关闭通道
	go func() {
		wg.Wait()
		close(results)
	}()

	// 主 goroutine 收集并处理结果
	fmt.Println("开始并行获取数据...")
	allResults := make([]Result, 0, len(sources))
	for result := range results {
		fmt.Printf("收到来自 %s 的数据: %s\n", result.Source, result.Data)
		allResults = append(allResults, result)
	}

	fmt.Printf("\n共获取到 %d 条数据，准备进行后续处理\n", len(allResults))
}

// fetchFromSource 模拟从不同数据源获取数据的函数
func fetchFromSource(source string) string {
	// 这里只是模拟
	return fmt.Sprintf("来自%s的数据内容", source)
}
```

程序说明：

> 使用带缓冲的通道来收集结果，这样可以避免 goroutine 在发送数据时阻塞；
>
> 使用闭包来传递循环变量，确保每个 goroutine 处理正确的数据源；
>
> 启动一个单独的 goroutine 来等待所有任务完成后关闭通道，让主 goroutine 可以安全地遍历通道中的所有数据。

程序运行结果：

> go run ./main.go
>
> 开始并行获取数据...
> 收到来自 外部API 的数据: 来自外部API的数据内容
> 收到来自 用户数据库 的数据: 来自用户数据库的数据内容
> 收到来自 Redis缓存 的数据: 来自Redis缓存的数据内容
>
> 共获取到 3 条数据，准备进行后续处理

### 场景2：并发HTTP请求

http2.go

```go
package main

import (
    "fmt"
    "io"
    "net/http"
    "sync"
    "time"
)

func fetchURL(url string, wg *sync.WaitGroup, results chan<- string) {
    defer wg.Done()
    
    client := http.Client{Timeout: 5 * time.Second}
    resp, err := client.Get(url)
    if err != nil {
        results <- fmt.Sprintf("%s: 错误 - %v", url, err)
        return
    }
    defer resp.Body.Close()
    
    body, _ := io.ReadAll(resp.Body)
    results <- fmt.Sprintf("%s: 状态码 %d, 大小 %d bytes", 
        url, resp.StatusCode, len(body))
}

func main() {
    urls := []string{
        "https://www.baidu.com",
        "https://www.so.com",
    }
    
    var wg sync.WaitGroup
    // 用 channel 来存结果
    results := make(chan string, len(urls))
    
    for _, url := range urls {
        wg.Add(1)
        go fetchURL(url, &wg, results)
    }
    
    // 等待所有请求完成
    wg.Wait()
    close(results)
    
    // 输出结果
    for result := range results {
        fmt.Println(result)
    }
}
```

### 场景3：并发文件处理

```go
package main

import (
    "bufio"
    "fmt"
    "os"
    "sync"
)

func main() {
    var wg sync.WaitGroup
    files := []string{"data1.txt", "data2.txt", "data3.txt"}

    for _, f := range files {
        // 1. 在启动 goroutine 之前 Add
        wg.Add(1)
        go processFile(f, &wg)
    }

    // 3. 阻塞等待所有任务完成
    wg.Wait()
    fmt.Println("所有文件处理完毕")
}

func processFile(filename string, wg *sync.WaitGroup) {
    // 2. 确保在函数退出时调用 Done
    defer wg.Done()

    file, err := os.Open(filename)
    if err != nil {
        fmt.Printf("Error opening %s: %v\n", filename, err)
        return
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    lineCount := 0
    for scanner.Scan() {
        lineCount++
    }
    fmt.Printf("File: %s, Lines: %d\n", filename, lineCount)
}
```

上面的 2 个场景代码 并发 http 请求和并发文件处理程序，程序差不多，启动多个 goroutine 处理 http和文件，然后用 WaitGroup 来控制 goroutine。

### 场景4：工作池模式(限制并发数量)

workpool.go

```go
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
```

程序说明：

> 这个 worker 池实现展示了 WaitGroup 在复杂并发场景中的应用。
>
> 关键点：
>
> 每个 worker 在启动时调用 Add 方法，在退出时通过 defer 调用 Done 方法；
>
> 主线程通过调用 Stop 方法先关闭任务通道，然后等待所有 worker 完成后再继续。
>
> 需要注意的是，关闭通道后，所有 worker 的 for-range 循环会自动退出，这是 Go channel 的一个重要特性，让 worker 的优雅退出成为可能。

程序运行：

> gogo ❯❯ go run ./workpool.go
>
> Worker 2 开始处理任务 3
> Worker 0 开始处理任务 1
> Worker 1 开始处理任务 2
> Worker 1 完成处理任务 2
> Worker 1 开始处理任务 4
> Worker 2 完成处理任务 3
> Worker 2 开始处理任务 5
> Worker 0 完成处理任务 1
> Worker 0 开始处理任务 6
> Worker 2 完成处理任务 5
> Worker 1 完成处理任务 4
> Worker 0 完成处理任务 6
> 所有worker已停止
> 程序正常退出

## 使用WaitGroup最佳实践

### 正确实践

#### 1. 使用 defer 调用 Done()

```go
func goodPractice(wg *sync.WaitGroup) {
    defer wg.Done() // 确保函数退出时一定会调用
    
    // 业务逻辑
    // 即使发生 panic，defer 也会执行
}
```

#### 2. 在启动 goroutine 前调用 Add()

基本原则是：Add 方法必须在启动 goroutine 之前调用，或者至少在 goroutine 内部但在任何可能调用 Wait 方法之前调用。



```go
func correctOrder() {
    var wg sync.WaitGroup
    
    for i := 0; i < 10; i++ {
        wg.Add(1) // 先 Add
        go func(id int) {
            defer wg.Done()
            fmt.Println(id)
        }(i)
    }
    
    wg.Wait()
}
```

#### 3. 通过指针传递 WaitGroup

```go
func processWithWaitGroup(wg *sync.WaitGroup) { // 使用指针
    defer wg.Done()
    // 处理逻辑
}

func main() {
    var wg sync.WaitGroup
    wg.Add(1)
    go processWithWaitGroup(&wg) // 传递指针
    wg.Wait()
}
```

#### 4. 结合 context 使用

```go
package main

import (
    "context"
    "fmt"
    "sync"
    "time"
)

func workerWithContext(ctx context.Context, id int, wg *sync.WaitGroup) {
    defer wg.Done()
    
    for {
        select {
        case <-ctx.Done():
            fmt.Printf("Worker %d 收到取消信号\n", id)
            return
        default:
            fmt.Printf("Worker %d 工作中...\n", id)
            time.Sleep(time.Millisecond * 500)
        }
    }
}

func main() {
    var wg sync.WaitGroup
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    
    for i := 1; i <= 3; i++ {
        wg.Add(1)
        go workerWithContext(ctx, i, &wg)
    }
    
    wg.Wait()
    fmt.Println("所有 worker 已停止")
}
```

### 错误实践

#### 错误1：在 goroutine 内部调用 Add()

```go
// 错误：可能导致 Wait() 过早返回
func badPractice1() {
    var wg sync.WaitGroup
    
    for i := 0; i < 10; i++ {
        go func(id int) {
            wg.Add(1) // 错误：在 goroutine 内部 Add
            defer wg.Done()
            fmt.Println(id)
        }(i)
    }
    
    wg.Wait() // 可能在所有 goroutine Add 之前就返回
}

// 正确：在启动 goroutine 前 Add
func goodPractice1() {
    var wg sync.WaitGroup
    
    for i := 0; i < 10; i++ {
        wg.Add(1) // 正确：在启动前 Add
        go func(id int) {
            defer wg.Done()
            fmt.Println(id)
        }(i)
    }
    
    wg.Wait()
}
```

#### 错误2：复制 WaitGroup

```go
// 错误：复制 WaitGroup 会导致计数器不同步
func badPractice2(wg sync.WaitGroup) { // 值传递，会复制
    defer wg.Done() // 操作的是副本
    fmt.Println("这不会正确同步")
}

func main() {
    var wg sync.WaitGroup
    wg.Add(1)
    go badPractice2(wg) // 传递副本
    wg.Wait() // 永远等待
}

// 正确：使用指针
func goodPractice2(wg *sync.WaitGroup) {
    defer wg.Done()
    fmt.Println("正确同步")
}
```

#### 错误3：Add 和 Done 不匹配

```go
// 错误：计数器不匹配
func badPractice3() {
    var wg sync.WaitGroup
    
    wg.Add(2) // Add 2 次
    
    go func() {
        defer wg.Done()
        fmt.Println("Task 1")
    }()
    
    // 只有一个 Done()，计数器永远不会归零
    wg.Wait() // 死锁！
}

// 正确：确保 Add 和 Done 匹配
func goodPractice3() {
    var wg sync.WaitGroup
    
    wg.Add(2)
    
    go func() {
        defer wg.Done()
        fmt.Println("Task 1")
    }()
    
    go func() {
        defer wg.Done()
        fmt.Println("Task 2")
    }()
    
    wg.Wait()
}
```

#### 错误4：重复使用 WaitGroup 不当

```go
// 错误：在 Wait() 返回前重新使用
func badPractice4() {
    var wg sync.WaitGroup
    
    for round := 0; round < 3; round++ {
        wg.Add(1)
        go func(r int) {
            defer wg.Done()
            fmt.Printf("Round %d\n", r)
        }(round)
    }
    
    wg.Wait() // 等待第一批完成
    
    // 危险：如果上面的 goroutine 还在执行，这里可能导致问题
}

// 正确：每轮都正确等待
func goodPractice4() {
    for round := 0; round < 3; round++ {
        var wg sync.WaitGroup // 每轮创建新的 WaitGroup
        
        wg.Add(1)
        go func(r int) {
            defer wg.Done()
            fmt.Printf("Round %d\n", r)
        }(round)
        
        wg.Wait()
    }
}
```

#### 错误5：负数或过大的计数器

```go
// 错误：调用 Done() 次数超过 Add()
func badPractice5() {
    var wg sync.WaitGroup
    
    wg.Add(1)
    wg.Done()
    wg.Done() // panic: sync: negative WaitGroup counter
}

// 正确：严格匹配
func goodPractice5() {
    var wg sync.WaitGroup
    
    wg.Add(1)
    go func() {
        defer wg.Done()
        // 工作逻辑
    }()
    
    wg.Wait()
}
```

## 性能优化建议

- 1. **批量 Add**: 在循环外一次性 Add 多个计数，而不是循环内每次 Add(1)

```go
var wg sync.WaitGroup
wg.Add(100) // 一次性添加

for i := 0; i < 100; i++ {
    go func() {
        defer wg.Done()
        // 工作
    }()
}
```

- 2. **避免过多 goroutine**: 使用工作池模式限制并发数量

     

- 3. **结合 channel 使用**: WaitGroup 用于等待完成，channel 用于传递数据

## 总结

`sync.WaitGroup` 是 Go 并发编程中的基础工具，正确使用它需要注意：

- 始终通过指针传递
- 在 goroutine 启动前调用 Add()
- 使用 defer 确保 Done() 被调用
- 保持 Add() 和 Done() 调用次数匹配
- 避免复制 WaitGroup 值

掌握上面的一些使用最佳实践，就能安全高效地协调并发任务的执行。