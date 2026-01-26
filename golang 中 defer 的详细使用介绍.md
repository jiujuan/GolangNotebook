## defer 介绍

在平常的编程中，经常需要在函数结束时执行一些清理工作，比如关闭文件、释放锁、关闭数据库连接等。传统做法，程序员必须在函数的每个返回点之前手动调用清理函数，这不仅繁琐，而且极易出错。

想象一个场景：

> 如果你的函数有多个返回路径（正常返回、错误返回、提前返回），在每个路径都正确地调用清理代码将成为一场噩梦。遗漏任何一处都会导致资源泄漏，而这种泄漏往往在系统运行数小时甚至数天后才暴露出来，其排查难度极高。

Go 编程语言中的 `defer`关键字用于延迟函数调用的执行，使用 defer 的函数会在包含它的函数返回之前执行，无论函数是正常返回还是发生 panic。defer 确保资源清理代码一定会被执行。

defer 是一个后进先出的结构，多个 defer 语句按照栈的方式执行，最后声明的最先执行。defer 语句执行是在 return 语句之后，函数真正返回之前执行。

## 基本使用

defer 基本用法：defer func()

defer/basic.go

```go
package main

import "fmt"

func basicExample() {
    fmt.Println("函数开始执行")
    
    // 声明第一个defer
    defer fmt.Println("第一个defer执行")
    
    fmt.Println("函数中间执行")
    
    // 声明第二个defer，最后声明的 defer 先执行
    defer fmt.Println("第二个defer执行")
    
    fmt.Println("函数即将结束")
}

func main() {
    basicExample()
    // 输出顺序：
    // 函数开始执行
    // 函数中间执行
    // 函数即将结束
    // 第二个defer执行
    // 第一个defer执行
}
```

## 常见使用常见

### 场景1：关闭文件

defer/close_file.go

```go
package main

import (
    "fmt"
    "os"
)

func writeFile(filename string) error {
    file, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer file.Close() // 确保文件被关闭
    
    _, err = file.WriteString("Hello, defer!")
    if err != nil {
        return err
    }
    
    return nil
}

func main() {
    filename := "./a.txt"
    writeFile(filename)
    fmt.Println("End!")
}
```

这个示例展示了`defer`的第一个重要特性：**后进先出（LIFO, Last In First Out）**。最后声明的`defer`会最先执行，就像叠盘子一样，最后放上去的盘子会最先被取下来。

### 场景2：释放互斥锁

defer/close_mutex.go

```go
package main

import (
    "fmt"
    "sync"
)

type SafeCounter struct {
    mu    sync.Mutex
    count int
}

func (c *SafeCounter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock() // 确保锁被释放
    
    c.count++
    
}

func (c *SafeCounter) GetCount() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    return c.count
}

func main() {
    sc := &SafeCounter{}
    sc.Increment()
    fmt.Println(sc.GetCount())
}
```

### 场景3：关闭数据库连接

```go
func dbOperation() {
    db, err := sql.Open("mysql", "user:password@/dbname")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close() // 确保函数结束时关闭连接

    // 进行数据库操作...
}
```

### 场景4：与recover实现异常捕获和处理

捕获异常，然后用 defer 结合 recover 来处理异常

defer/recover.go

```go
package main

import (
    "fmt"
    "log"
)

func safeFunc() {
    // 必须在 panic 之前 defer
    defer func() {
        if r := recover(); r != nil {
            fmt.Println("捕获到异常:", r) // 异常被处理，程序不崩溃
            log.Println("发生panic，已恢复")
        }
    }()

    fmt.Println("执行正常逻辑...")
    panic("发生严重错误") // 抛出异常
    fmt.Println("这行代码不会被执行")
}

func main() {
    safeFunc()
    fmt.Println("程序正常结束")
}

/**
 运行程序： go run ./recover.go

 输出：

执行正常逻辑...
捕获到异常: 发生严重错误
2026/01/26 23:44:08 发生panic，已恢复
程序正常结束 
 **/
```

### 场景5：记录函数耗时

defer/record_time.go

```go
package main

import (
	"fmt"
	"time"
)

func main() {
	// 调用被测试函数
	doSomething()
}

func doSomething() {
	// 1. 在函数开头记录开始时间
	start := time.Now()

	// 2. 使用 defer 注册一个匿名函数，在函数退出时计算耗时
	defer func() {
		// 3. 计算耗时
		elapsed := time.Since(start)
		fmt.Printf("函数执行耗时: %s\n", elapsed)
	}()

	// 4. 模拟函数实际逻辑（例如：休眠 100 毫秒）
	time.Sleep(100 * time.Millisecond)
	fmt.Println("任务完成")
}
```