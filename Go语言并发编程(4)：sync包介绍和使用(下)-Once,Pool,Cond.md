sync包下：Once，Pool，Cond

## 一、sync.Once 执行一次

### Once 简介

- sync.Once 是 Go 提供的让函数只执行一次的一种实现。
- 如果 once.Do(f) 被调用多次，只有第一次调用会调用 f。

常用场景：

- 用于单例模式，比如初始化数据库配置

Once 提供的方法：

- 它只提供了一个方法 `func (o *Once) Do(f func())`

### 例 1，基本使用：

```go
package main

import (
	"fmt"
	"sync"
)

func main() {
	var once sync.Once

	func1 := func() {
		fmt.Println("func1")
	}
	once.Do(func1)

	func2 := func() {
		fmt.Println("func2")
	}
	once.Do(func2)
}
```

运行输出：

func1

多次调用 Once.Do() 只会执行第一次调用。

## 二、sync.Pool 复用对象

### Pool 简介

sync.Pool 可以单独保存和复用临时对象，可以认为是一个存放对象的临时容器或池子。也就是说 Pool 可以临时管理多个对象。

存储在 Pool 中的对象都可能随时自动被 GC 删除，也不会另行通知。所以它不适合像 socket 长连接或数据库连接池。

一个 Pool 可以安全地同时被多个 goroutine 使用。

Pool 的目的是缓存已分配内存但未使用的对象供以后使用(重用)，减轻 GC 的压力，后面使用也不用再次分配内存。它可以构建高效、线程安全的空闲列表。



主要用途：

> - Pool 可以作为一个临时存储池，把对象当作一个临时对象存储在池中，然后进行存或取操作，这样对象就可以重用，不用进行内存分配，减轻 GC 压力



Pool 的数据结构和 2 方法 Get() 和 Put()：

```go
// https://pkg.go.dev/sync@go1.20#Pool
type Pool struct {
    ...
    
	New func() any
}

func (p *Pool) Get() any

func (p *Pool) Put(x any)
```

- Pool struct，里面的 New 函数类型，声明一个对象池
- Get() 从对象池中获取对象
- Put() 对象使用完毕后，返回到对象池里

### 例子1，基本使用

```go
package main

import (
	"fmt"
	"sync"
)

func main() {
    // 创建一个 Pool
	pool := sync.Pool{
        // New 函数用处：当我们从 Pool 中用 Get() 获取对象时，如果 Pool 为空，则通过 New 先创建一个
        // 对象放入 Pool 中，相当于给一个 default 值
		New: func() interface{} {
			return 0
		},
	}

	pool.Put("lilei")
	pool.Put(1)

	fmt.Println(pool.Get())
	fmt.Println(pool.Get())
	fmt.Println(pool.Get())
	fmt.Println(pool.Get())
}

/** output：
lilei
1
0
0
**/
```

### 例子2，缓存临时对象

from：https://geektutu.com/post/hpg-sync-pool.html 极客兔兔上的一个例子

```go
package main

import "encoding/json"

type Student struct {
	Name   string
	Age    int32
	Remark [1024]byte
}

var buf, _ = json.Marshal(Student{Name: "Geektutu", Age: 25})

func unmarsh() {
	stu := &Student{}
	json.Unmarshal(buf, stu)
}
```

json 的反序列化的文本解析和网络通信，当程序在高并发下，需要创建大量的临时对象。这些对象又是分配在堆上，会给 GC 造成很大压力，会严重影响性能。这时候 sync.Pool 就派上用场了。而且 Pool 大小是动态可伸缩的，高负载会动态扩容。

使用 sync.Pool

```go
// 创建一个临时对象池
var studentPool = sync.Pool{
	New: func() interface{} {
		return new(Student)
	},
}

// Get 和 Set 操作
func unmarshByPool() {
	stu := studentPool.Get().(*Student) // Get 获取对象池中的对象，返回值是 interface{}，需要类型转换
	josn.Unmarshal(buf, stu)
	studentPool.Put(stu) // Put 对象使用完毕，返还给对象池
}
```

### 例子3，标准库 fmt.Printf

Go 语言标准库也大量使用了 `sync.Pool`，例如 `fmt` 和 `encoding/json`。

以下是 `fmt.Printf` 的源代码(go/src/fmt/print.go)：

```go
// https://github.com/golang/go/blob/release-branch.go1.20/src/fmt/print.go#L120
type pp struct {
	buf buffer

	// arg holds the current item, as an interface{}.
	arg any

	// value is used instead of arg for reflect values.
	value reflect.Value

	// fmt is used to format basic items such as integers or strings.
	fmt fmt
	... ...
}

var ppFree = sync.Pool{
	New: func() any { return new(pp) },
}

func newPrinter() *pp {
	p := ppFree.Get().(*pp)
	p.panicking = false
	p.erroring = false
	p.wrapErrs = false
	p.fmt.init(&p.buf)
	return p
}

func (p *pp) free() {
	if cap(p.buf) > 64*1024 {
		p.buf = nil
	} else {
		p.buf = p.buf[:0]
	}
	if cap(p.wrappedErrs) > 8 {
		p.wrappedErrs = nil
	}

	p.arg = nil
	p.value = reflect.Value{}
	p.wrappedErrs = p.wrappedErrs[:0]
	ppFree.Put(p)
}

func (p *pp) Write(b []byte) (ret int, err error) {
	p.buf.write(b)
	return len(b), nil
}

func (p *pp) WriteString(s string) (ret int, err error) {
	p.buf.writeString(s)
	return len(s), nil
}

func Fprintf(w io.Writer, format string, a ...any) (n int, err error) {
	p := newPrinter()
	p.doPrintf(format, a)
	n, err = w.Write(p.buf)
	p.free()
	return
}
```

## 三、sync.Cond 条件变量

### Cond 简介

Cond 用互斥锁和读写锁实现了一种条件变量。

那什么是条件？

比如在 Go 中，某个 goroutine 协程只有满足了一些条件的情况下才能执行，否则等待。

比如并发中的协调共享资源情况，共享资源状态发生了变化，在程序中可以看作是某种条件发生了变化，在锁上等待的 goroutine，就可以通知它们，“你们要开始干活了”。

那怎么通知？

Go 中的 `sync.Cond` 在锁的基础上增加了一个消息通知的功能，保存了一个 goroutine 通知列表，用来唤醒一个或所有因等待条件变量而阻塞的 goroutine。它这个通知列表实际就是一个等待队列，队列里存放了所有因等待条件变量(sync.Cond)而阻塞的 goroutine。

我们看下 sync.Cond 的数据结构：

```go
// https://github.com/golang/go/blob/release-branch.go1.19/src/sync/cond.go#L36
type Cond struct {
	noCopy noCopy

	// L is held while observing or changing the condition
	L Locker

	notify  notifyList
	checker copyChecker
}
// https://cs.opensource.google/go/go/+/refs/tags/go1.19:src/sync/runtime2.go;drc=ad461f3261d755ab24222bc8bc30624e03646c3b;l=13
type notifyList struct {
	wait   uint32 // 下一个等待唤醒的 goroutine 索引，在锁外自动增加
	notify uint32 // 下一个要通知的 goroutine 索引，只能在持有锁的情况下写入，读取可以不要锁
	lock   uintptr // key field of the mutex
	head   unsafe.Pointer // 链表头
	tail   unsafe.Pointer // 链表尾
}
```

变量 notify 就是通知列表。

> sync.Cond 用来协调那些访问共享资源的 goroutine，当共享资源条件发生变化时，sync.Cond 就可以通知那些等待条件发生而阻塞的 goroutine。

既然是通知 goroutine 的功能，那与 channel 作为通知功能有何区别？

### 与 channel 的区别

举个例子，在并发编程里，多个协程工作的程序，有一个协程 g1 正在接收数据，其它协程必须等待 g1 执行完，才能开始读取到正确的数据。当 g1 接收完成后，怎么通知其它所有协程？说：我读完了，你们开始干活了(开始读取数据)。

想一想，用互斥锁或channel？它们一般只能控制一个协程可以等待并读取数据，并不能很方便的通知其它所有协程。

还有其它方法么？想到的第一个方法，主动去问：

- 给 g1 一个全局变量，用来标识是否接收完，其它协程反复检查该变量看是否接收完。

第二个方法，被动等通知，其它所有协程等通知：

- 其它协程阻塞，g1 接收完毕后，通知其它协程。 这个阻塞可以是给每一个协程一个 channel 进行阻塞，g1 接收完，通知每一个 channel 解除阻塞。

(上面2种情况，让我想到了网络编程中的 select 和 epoll 的优化，select 不断轮询看数据是否接收完，epoll 把 socket 的读和写看作是事件，读完了后主动回调函数进行处理。这个少了通知直接调用回调函数处理)



遇到这种情况，Go 给出了它的解决方法 - sync.Cond，就可以解决这个问题。它可以广播唤醒所有等待的 goroutine。

sync.Cond 有一个唤醒列表，Broadcast 通过这个列表通知所有协程。

### sync.Cond 使用情况总结

> 1、多个 goroutine 阻塞等待，一个 goroutine 通知所有，这时候用 sync.Cond。一个生产者，多个消费者
>
> 2、一个 goroutien 阻塞等待，一个 goroutine 通知一个，这时候用 锁 或 channel

### sync.Cond 的方法

从官网 https://pkg.go.dev/sync@go1.19#Cond 可以看出，有 4 个方法，分别是 NewCond()，Broadcast()，Signal()，Wait()。

> - NewCond：创建一个 sync.Cond 变量
> - Broadcast：广播唤醒所有 wait 的 goroutine
> - Signal：一次只唤醒一个，哪个？最优先等待的 goroutine
> - Wait：等待条件唤醒

- NewCond() 创建 Cond 实例

```go
// https://github.com/golang/go/blob/release-branch.go1.19/src/sync/cond.go#L46
// NewCond returns a new Cond with Locker l.
func NewCond(l Locker) *Cond {
	return &Cond{L: l}
}
```

从上面方法可以看出，NewCond 创建实例需要传入一个锁，sync.NewCond(&sync.Mutex{})，返回一个带有锁的新 Cond。

- BroadCast() 广播唤醒所有

```go
// https://github.com/golang/go/blob/release-branch.go1.19/src/sync/cond.go#L90
// Broadcast wakes all goroutines waiting on c.
//
// It is allowed but not required for the caller to hold c.L
// during the call.
func (c *Cond) Broadcast() {
	c.checker.check()
	runtime_notifyListNotifyAll(&c.notify)
}
```

广播唤醒所有等待在条件变量 c 上的 goroutines。

- Signal() 信号唤醒一个协程

```go
// https://github.com/golang/go/blob/release-branch.go1.19/src/sync/cond.go#L81
// Signal wakes one goroutine waiting on c, if there is any.
//
// It is allowed but not required for the caller to hold c.L
// during the call.
//
// Signal() does not affect goroutine scheduling priority; if other goroutines
// are attempting to lock c.L, they may be awoken before a "waiting" goroutine.
func (c *Cond) Signal() {
	c.checker.check()
	runtime_notifyListNotifyOne(&c.notify)
}
```

信号唤醒等待在条件变量 c 上的一个 goroutine。

- Wait() 等待

```go
// https://github.com/golang/go/blob/release-branch.go1.19/src/sync/cond.go#L66
// Wait atomically unlocks c.L and suspends execution
// of the calling goroutine. After later resuming execution,
// Wait locks c.L before returning. Unlike in other systems,
// Wait cannot return unless awoken by Broadcast or Signal.
//
// Because c.L is not locked when Wait first resumes, the caller
// typically cannot assume that the condition is true when
// Wait returns. Instead, the caller should Wait in a loop:
//
//	c.L.Lock()
//	for !condition() {
//	    c.Wait()
//	}
//	... make use of condition ...
//	c.L.Unlock()
func (c *Cond) Wait() {
	c.checker.check()
	t := runtime_notifyListAdd(&c.notify)
	c.L.Unlock()
	runtime_notifyListWait(&c.notify, t)
	c.L.Lock()
}
```

Wait() 用于阻塞调用者，等待通知。向 notifyList 注册一个通知，然后阻塞等待被通知。

看上面代码：

> `runtime_notifyListAdd()` 将当前 go 程添加到通知列表，等待通知
>
> `runtime_notifyListWait()` 将当前 go 程休眠，接收到通知后才被唤醒



对条件的检查，使用了 for !condition() 而非 if，是因为当前协程被唤醒时，条件不一定符合要求，需要再次 Wait 等待下次被唤醒。为了保险起见，使用 for 能够确保条件符合要求后，再执行后续的代码。

```go
c.L.Lock()
for !condition() {
    c.Wait()
}
... make use of condition ...
c.L.Unlock()
```

### 例子1

来自：https://stackoverflow.com/questions/36857167/how-to-correctly-use-sync-cond 的一个例子

```go
package main

import (
	"fmt"
	"sync"
)

// https://stackoverflow.com/questions/36857167/how-to-correctly-use-sync-cond
var sharedRsc = make(map[string]interface{})

func main() {
	var wg sync.WaitGroup
	wg.Add(2)

	fmt.Println("process start, sharedRsc len: ", len(sharedRsc))

	mutex := sync.Mutex{}
	cond := sync.NewCond(&mutex)

	go func() {
		// this go routine wait for changes to the sharedRsc
		cond.L.Lock()
		for len(sharedRsc) == 0 { // 条件为0，cond.Wait() 阻塞当前goroutine，并等待通知
			cond.Wait()
		}

		fmt.Println("sharedRsc[res1]:", sharedRsc["res1"])
		cond.L.Unlock()
		wg.Done()
	}()

	go func() {
		// this go routine wait for changes to the sharedRsc
		cond.L.Lock()
		for len(sharedRsc) == 0 { // 条件为0，cond.Wait() 阻塞当前goroutine，并等待通知
			cond.Wait()
		}
		fmt.Println("sharedRsc[res2]:", sharedRsc["res2"])
		cond.L.Unlock()
		wg.Done()
	}()

	// this one writes changes to sharedRsc
	cond.L.Lock()
	sharedRsc["res1"] = "one"
	sharedRsc["res2"] = "two"
	cond.Broadcast() // 通知所有获取锁的 goroutine
	cond.L.Unlock()

	wg.Wait()

	fmt.Print("process end!!!")
}
/**
作者：garbagecollector
Having said that, using channels is still the recommended way to pass data around if the situation permitting.
作者建议：如果条件允许，channel 还是最好的数据通信方式
Note: sync.WaitGroup here is only used to wait for the goroutines to complete their executions.
**/
```

## 四、参考

- https://www.cnblogs.com/qcrao-2018/p/12736031.html 深度解密 Go 语言之 sync.Pool，作者：Stefno
- https://geektutu.com/post/hpg-sync-pool.html Go sync.Pool 复用对象，作者：极客兔兔
- https://pkg.go.dev/sync#Pool 
- https://cs.opensource.google/go/go/+/refs/tags/go1.19:src/runtime/sema.go;l=482
- https://stackoverflow.com/questions/36857167/how-to-correctly-use-sync-cond