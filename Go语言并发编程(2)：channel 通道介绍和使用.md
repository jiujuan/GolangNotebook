## 一、简介

channel 不仅可以用于 goroutine 间进行安全通信，还可以用于同步内存访问。

而且 Go 社区强烈推荐使用 channel 通道实现 goroutine 之间的通信，

> 不要通过共享内存来通信，而应该通过通信来共享内存。

Go 从语言层面保证了同一时间只有一个 goroutine 能够访问 channel 里的数据，从而保证数据安全。

在 Go 中使用 channel 通信，通过通信来传递内存数据，让内存数据可以在不同的 goroutine 之间进行传递，而不是

用共享内存来通信。

## 二、channel 的基本使用

前面有 2 篇文章介绍 chanel 的使用：

- https://www.cnblogs.com/jiujuan/p/11723586.html goroutine 协程和 channel 通道

- https://www.cnblogs.com/jiujuan/p/16014608.html golang 中 channel 的详细使用、使用注意事项及死锁分析

还有 1 篇讲 channel 的原理：

- https://www.cnblogs.com/jiujuan/p/12026551.html 深入理解Go语言：channel原理

## 三、channel 使用注意事项

### 关闭 channel

- 一般由发送端关闭 channel
- 向一个已经关闭的 channel 发送数据，会 panic
- 读取关闭的 channel ，返回零值

### nil channel

- 读取一个 nil channel，操作将阻塞

所以需要阻塞时，你可以手动修改 channel 为 nil，就会出现阻塞效果。

### for...range... 遍历channel

当 for range 遍历 channel 时，如果发送者没有关闭 channel 或在 range 之后关闭，会导致死锁。