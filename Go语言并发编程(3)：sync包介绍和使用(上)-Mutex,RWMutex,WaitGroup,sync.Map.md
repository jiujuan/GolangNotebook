## 一、sync 包简介

在并发编程中，为了解决竞争条件问题，Go 语言提供了 sync 标准包，它提供了基本的同步原语，例如互斥锁、读写锁等。

sync 包使用建议：

> 除了 Once 和 WaitGroup 类型之外，大多数类型旨在供低级库程序使用。更高级别的同步最好用 channel 通道和通信来完成。

sync 包中类型：

- sync.Mutex 互斥锁
- sync.RWMutex 读写锁
- sync.WaitGroup 等待组(等待一组 goroutine 完成)
- sync.Map 并发 Map
- sync.Once 执行一次
- sync.Pool 对象池
- sync.Cond 条件变量

此外，sync 下还有一个包 atomic， 它提供了对数据的原子操作。

另外，Go 的扩展包也提供了信号量这种同步原语：

- x/sync/semaphore

## 二、sync.Mutex 互斥锁

`sync.Mutex` 是一个互斥锁，它的作用就是保护临界区，确保同一时间只有一个 Go 协程进入临界区。

什么是临界区？为什么有临界区？

> 在并发编程中，有一部分程序被并发访问，这个访问可能是多个协程/线程修改这部分程序数据，这样的操作会导致意想不到的结果，为了不让操作导致意外结果，怎么办？就需要把这部分程序保护起来，一次只允许一个协程/线程访问这部分区域。需要被保护的这部分程序区域就叫临界区。
>
> 防止多个协程/线程同时进入临界区，修改程序数据。

互斥锁就是一种可以保护临界区资源方式。

互斥锁其实是一种最特殊的信号量，这个"量"只有 0 和 1，所以也叫互斥量。互斥量的值为 0 和 1，用来表示加锁和解锁。互斥锁是一种独占锁，即同一时间只能有一个协程持有锁，其他协程必须等待。

互斥锁使得同一时刻只有一个协程执行某段程序，其他协程等待该协程执行完在抢锁后执行。

![image-20230302144218909](https://img2023.cnblogs.com/blog/650581/202303/650581-20230324124458988-1667266600.png)

如上图所示：g1 用互斥锁保护临界区，g2 在中间尝试获取锁失败，g1 离开临界区释放锁，g2 获取到锁然后进行相应操作，操作完后释放锁离开临界区。



> 第一次使用后不得复制 Mutex。



互斥锁使用：

- 互斥锁有两个方法 `Lock()` 加锁和 `Unlock()` 解锁，他们是成对出现。当一个协程对资源上锁后，其他协程只能等待该协程解锁之后，才能再次上锁。
- 它还有一个 `TryLock()`，go1.18 之后添加的。
  - 当一个 goroutine 调用此方法试图获取锁时，如果这把锁没有被其他 goroutine 持有，那么这个 goroutine 获取锁并返回 true；
  - 如果这把锁已经被其它 goroutine 持有，或正准备给某个唤醒的 gorouine，那么请求锁的 goroutine 直接返回 false，不会阻塞在方法调用上。

```go
Lock()
代码段(临界区)
Unlock()
```



> 为了防止上锁后忘记释放锁，实际使用中用 defer 来释放锁。



例子：

```go
package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	var a = 0

	var lock sync.Mutex
	for i := 0; i < 100; i++ { // 并发 100 个goroutine
		go func(id int) {
			lock.Lock()
			defer lock.Unlock()
			a += 1
			fmt.Printf("goroutine %d, a=%d\n", id, a)
		}(i)
	}

	time.Sleep(time.Second) //等待1秒， 确保所有的协程执行完
}
```

## 三、sync.RWMutex 读写锁

sync.RWMutex 读写锁，对数据操进加锁进一步细分，针对读操作和写操作分别进行加锁和解锁。

> 在读写锁下，读操作和读操作之间不互斥，多个写操作是互斥，读操作和写操作也是互斥。



- 当一个 goroutine 获取读锁之后，其它的 goroutine 此时想获取读锁，那么可以继续获取锁，不用等待解锁；此时想获取写锁，就会阻塞等待直到读解锁；
- 当一个 goroutine 获取写锁之后，其它的 goroutine 无论是获取读锁还是写锁，都会阻塞等待。



读写锁的好处：

> 多个读之间不互斥，读锁就可以降低对数据读取加互斥锁的性能损耗。而不像互斥锁那样对所有的数据操作，不管是读还是写，同等对待，都加一把大锁处理。
>
> 在读多写少的场景下，更适合用读写锁。



RWMutex 读写锁的方法：

> - Mutex 的加锁和解锁：Lock() 和 Unlock()
> - 只读加锁和加锁：RLock() 和 RUnlock()
>   - RLock() 加读锁时如果存在写锁，则不能加锁；当只有读锁或无锁时，可以加读锁，且读锁可以加载多个。
>   - RUnlock() 解读锁。没有读锁情况下调用 RUlock() 会导致 panic。
>
> 释放锁用 defer 来释放锁



```go
// 使用 RWMutex 的伪码，当然正式代码不会这样写，会用 defer 释放锁
mutex := sync.RWMutex{}

mutex.Lock()
// 操作的资源
mutex.Unlock()

mutex.RLock()
// 读的资源
mutex.RUlock()
```

例子：

```go
package main

import (
	"fmt"
	"sync"
	"time"
)

var sum = 0
var rwMutex sync.RWMutex

func main() {
	// 并发写
	for i := 1; i <= 50; i++ {
		go writeSum()
	}

	// 并发读
	for i := 1; i <= 20; i++ {
		go fmt.Println("readSum: ", readSum())
	}

	time.Sleep(time.Second * 2) // 防止主程序退出，子协程还没运行完
	fmt.Println("end sum: ", sum)
}

func writeSum() {
	rwMutex.Lock()         // 读写锁
	defer rwMutex.Unlock() // 释放锁
	sum += 1
}

func readSum() int {
	rwMutex.RLock()         // 读写锁加读锁
	defer rwMutex.RUnlock() // 释放读锁
	return sum
}
```

## 四、sync.WaitGroup 等待组

sync.WaitGroup，等待一组或多个 goroutine 执行完成。



WaitGroup 内部有一个安全的计数器，它调用 Add(n int) 方法把计数器 +n；使用 Done() 方法，将计数器减 1，Done() 的底层是调用 Add(-1)；调用 Wait() 方法等待所有的 goroutine 执行完，即计数器为 0，Wait() 就返回。

WaitGroup 详细原理，可以看我前面的文章：[sync.WaitGroup源码分析 ](https://www.cnblogs.com/jiujuan/p/16735012.html)。



- WaitGroup 里的方法：

> - Add(n)，设置要等待的子 goroutine 数量，n 表示要等待数量
> - Done()，子 goroutine 执行完后，计数器减一
> - Wait()，阻塞等待所有子 goroutine 执行完 

例子：

```go
package main

import (
	"fmt"
	"sync"
)

func main() {
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		fmt.Println("子 goroutine1")
	}()

	go func() {
		defer wg.Done()
		fmt.Println("子 goroutine2")
	}()

	wg.Wait() // 等待所有的子goroutine结束

	fmt.Println("程序运行结束")
}
```

## 五、sync.Map 并发Map

在 Go 语言中，内置数据结构 map 并不是并发安全的，所以官方就出了一个 sync.Map。

希望了解 sync.Map 的原理，可以看这篇文章：[深入理解Go语言(05)：sync.map原理分析 ](https://www.cnblogs.com/jiujuan/p/13365901.html)。



[sync.Map](https://pkg.go.dev/sync#Map) 里的常用方法：

> go v1.20.1
>
> - Store(key, value any)，设置键的值
> - Load(key any) (value any, ok bool)，获取值
> - Delete(key any)，根据 key 删除值
> - LoadAndDelete(key any) (value any, loaded bool)，根据 key 删除值，返回以前的值如果还存在
> - LoadOrStore(key, value any)(actual any, loaded bool)，先根据 key 查找 value，如果找到则返回原来的值，loaded 为 true；如果没有找到 key 对应的 value 值，则存在 key，value 值并将存储值返回，loaded 为 false
> - Range(f func(key, value any) bool)，遍历 sync.Map 的元素
>
> 更多方法请查看：https://pkg.go.dev/sync#Map

例子：

```go
package main

import (
	"fmt"
	"sync"
)

func main() {
	var syncmap sync.Map

	syncmap.Store("li", 12)
	syncmap.Store("han", "lu")
	syncmap.Store("mei", 34)

	fmt.Println(syncmap.Load("han"))

	// key 不存在
	val, ok := syncmap.LoadOrStore("lei", "lei")
	fmt.Println(val, ok)
	// key 存在
	val, ok = syncmap.LoadOrStore("han", "cunzai")
	fmt.Println(val, ok)

	syncmap.Delete("mei")

	syncmap.Range(func(k, v any) bool {
		fmt.Println("k-v: ", k, v)
		return true
	})
}
```



## 参考

- https://pkg.go.dev/sync sync 
- https://www.cnblogs.com/jiujuan/p/13365901.html  sync.Map 原理