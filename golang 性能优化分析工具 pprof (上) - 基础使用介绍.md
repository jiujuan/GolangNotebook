
- [golang 性能优化分析工具 pprof (上) - 基础使用介绍](https://www.cnblogs.com/jiujuan/p/14588185.html)
- [golang 性能优化分析工具 pprof (下) - web 服务分析](https://www.cnblogs.com/jiujuan/p/14598141.html)

## 一、golang 程序性能调优

### 在 golang 程序中，有哪些内容需要调试优化？

一般常规内容：
1. **cpu**：程序对cpu的使用情况 - 使用时长，占比等
2. **内存**：程序对cpu的使用情况 - 使用时长，占比，内存泄露等。如果在往里分，程序堆、栈使用情况
3. **I/O**：IO的使用情况 - 哪个程序IO占用时间比较长

golang 程序中：
1. **goroutine**：go的协程使用情况，调用链的情况
2. **goroutine leak**：goroutine泄露检查
3. **go dead lock**：死锁的检测分析
4. **data race detector**：数据竞争分析，其实也与死锁分析有关

上面是在 golang 程序中，性能调优的一些内容。

### 有什么方法工具调试优化 golang 程序？

比如 linux 中 cpu 性能调试，工具有 top，dstat，perf 等。

那么在 golang 中，有哪些分析方法？

**golang 性能调试优化方法：**
- **Benchmark**：**基准测试**，对特定代码的运行时间和内存信息等进行测试
- **Profiling**：**程序分析**，程序的运行画像，在程序执行期间，通过采样收集的数据对程序进行分析
- **Trace**：**跟踪**，在程序执行期间，通过采集发生的事件数据对程序进行分析

> profiling 和 trace 有啥区别？
> profiling 分析没有时间线，trace 分析有时间线。

在 golang 中，应用方法的工具呢？

这里介绍 pprof 这个 golang 工具，它可以帮助我们调试优化程序。
>它的最原始程序是 [gperftools](https://github.com/gperftools/gperftools) - https://github.com/gperftools/gperftools，golang 的 pprof 是从它而来的。

## 二、pprof 介绍

### 简介

pprof 是 golang 官方提供的性能调优分析工具，可以对程序进行性能分析，并可视化数据，看起来相当的直观。
当你的 go 程序遇到性能瓶颈时，可以使用这个工具来进行调试并优化程序。

本文将对下面 golang 中 2 个监控性能的包 pprof 进行运用：

- [runtime/pprof](https://golang.org/pkg/runtime/pprof/)：采集程序运行数据进行性能分析，一般用于后台工具型应用，这种应用运行一段时间就结束。

- [net/http/pprof](https://golang.org/pkg/net/http/pprof/)：对 runtime/pprof 的二次封装，一般是服务型应用。比如 web server ，它一直运行。这个包对提供的 http 服务进行数据采集分析。

上面的 pprof 开启后，每隔一段时间就会采集当前程序的堆栈信息，获取函数的 cpu、内存等使用情况。通过对采样的数据进行分析，形成一个数据分析报告。

pprof 以 [profile.proto](https://github.com/google/pprof/blob/master/proto/profile.proto) 的格式保存数据，然后根据这个数据可以生成可视化的分析报告，支持文本形式和图形形式报告。
profile.proto 里具体的数据格式是 [protocol buffers](https://developers.google.com/protocol-buffers)。

那用什么方法来对数据进行分析，从而形成文本或图形报告？

用一个命令行工具 `go tool pprof` 。

### pprof 使用模式

- Report generation：报告生成

- Interactive terminal use：交互式终端

- Web interface：Web 界面


## 三、runtime/pprof

### 使用前的准备工作

调试分析 golang 程序，要开启 profile 然后开始采样数据。
然后安装：`go get github.com/google/pprof` ，后面分析会用到。

采样数据的方式：

- **第 1 种**，在 go 程序中添加如下代码：
[StartCPUProfile](https://golang.org/pkg/runtime/pprof/#StartCPUProfile) 为当前 process 开启 CPU profiling 。
[StopCPUProfile](https://golang.org/pkg/runtime/pprof/#StopCPUProfile) 停止当前的 CPU profile。当所有的 profile 写完了后它才返回。
```
// 开启 cpu 采集分析：
pprof.StartCPUProfile(w io.Writer)

// 停止 cpu 采集分析：
pprof.StopCPUProfile()
```

[WriteHeapProfile](https://golang.org/pkg/runtime/pprof/#WriteHeapProfile) 把内存 heap 相关的内容写入到文件中
```
pprof.WriteHeapProfile(w io.Writer)
```

- **第 2 种**，在 benchmark 测试的时候

```sh
go test -cpuprofile cpu.prof -memprofile mem.prof -bench .
```

还有就是对 web 服务（http server） 数据的采集
```
go tool pprof $host/debug/pprof/profile
```

### 程序示例

> go version go1.13.9

#### 例子 1

我们用第 1 种方法，在程序中添加分析代码，demo.go :

```go
package main

import (
	"bytes"
	"flag"
	"log"
	"math/rand"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write mem profile to `file`")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close()

		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	var wg sync.WaitGroup
	wg.Add(200)

	for i := 0; i < 200; i++ {
		go cyclenum(30000, &wg)
	}

	writeBytes()

	wg.Wait()

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close()
		runtime.GC()

		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("cound not write memory profile: ", err)
		}
	}
}

func cyclenum(num int, wg *sync.WaitGroup) {
	slice := make([]int, 0)
	for i := 0; i < num; i++ {
		for j := 0; j < num; j++ {
			j = i + j
			slice = append(slice, j)
		}
	}
	wg.Done()
}

func writeBytes() *bytes.Buffer {
	var buff bytes.Buffer

	for i := 0; i < 30000; i++ {
		buff.Write([]byte{'0' + byte(rand.Intn(10))})
	}
	return &buff
}
```

编译程序、采集数据、分析程序：
1. 编译 demo.go
```
go build demo.go
```
2. 用 pprof 采集数据，命令如下：
```
./demo.exe --cpuprofile=democpu.pprof  --memprofile=demomem.pprof
```
> 说明：我是 win 系统，这个 demo 就是 demo.exe ，linux 下是 demo

3. 分析数据，命令如下：
```
go tool pprof democpu.pprof
```

go tool pprof 简单的使用格式为：`go tool pprof [binary] [source]`
- binary： 是应用的二进制文件，用来解析各种符号
- source： 表示 profile 数据的来源，可以是本地的文件，也可以是 http 地址
 
> 要了解 go tool pprof 更多命令使用方法，请查看文档：`go tool pprof --help`

> **注意**：
> 
>  获取的 Profiling 数据是动态获取的，如果想要获取有效的数据，需要保证应用或服务处于较大的负载中，比如正在运行工作中的服务，或者通过其他工具模拟访问压力。
> 否则如果应用处于空闲状态，比如 http 服务处于空闲状态，得到的结果可能没有任何意义。
> （后面会遇到这种问题，http 的 web 服务处于空闲状态，采集显示的数据为空）

分析数据，基本的模式有 2 种：

- 一个是**命令行交互分析**模式
- 一个是**图形可视化分析**模式

#### 命令行交互分析

##### A：命令行交互分析

1. 分析上面采集的数据，命令： `go tool pprof democpu.pprof`

![](https://img2020.cnblogs.com/blog/650581/202103/650581-20210329122110682-306537236.png)

| 字段  | 说明  |
| --- | --- |
|**Type**：  | 分析类型，这里是 cpu | 
|**Duration**：  | 程序执行的时长 | 

Duration 下面还有一行提示，这是交互模式（通过输入 help 获取帮助信息，输入 o 获取选项信息）。

可以看出，go 的 pprof 操作还有很多其他命令。

2. 输入 help 命令，出来很多帮助信息：

![](https://img2020.cnblogs.com/blog/650581/202103/650581-20210329122128925-308223199.png)

Commands 下有很多命令信息，text ，top 2个命令解释相同，输入这个 2 个看看：

3. 输入 top，text 命令
>top 命令：对函数的 cpu 耗时和百分比排序后输出

top后面还可以带参数，比如：  top15

![](https://img2020.cnblogs.com/blog/650581/202103/650581-20210329122145607-2007829916.png)

输出了相同的信息。

| 字段  | 说明  |
| --- | --- |
| flat  | 当前函数占用 cpu 耗时 |
| flat % | 当前函数占用 cpu 耗时百分比|
| sum%  | 函数占用 cpu 时间累积占比，从小到大一直累积到 100% |
| cum | 当前函数加上调用当前函数的函数占用 cpu 的总耗时 |
| %cum | 当前函数加上调用当前函数的函数占用 cpu 的总耗时占比 |

从字段数据我们可以看出哪一个函数比较耗费时间，就可以对这个函数进一步分析。
分析用到的命令是 `list` 。
> list 命令：可以列出函数最耗时的代码部分，格式：list 函数名

从上面采样数据可以分析出总耗时最长的函数是 `main.cycylenum`，用 `list cyclenum` 命令进行分析，如下图：

![](https://img2020.cnblogs.com/blog/650581/202103/650581-20210329122206105-453790225.png)

发现最耗时的代码是 62 行：`slice = append(slice, j)` ，这里耗时有 1.47s ，可以对这个地方进行优化。

这里耗时的原因，应该是 slice 的实时扩容引起的。那我们空间换时间，固定 slice 的容量，make([]int, num * num)

##### B：命令行下直接输出分析数据
在命令行直接输出数据，基本命令格式为：
```sh
go tool pprof <format> [options] [binary] <source>
```
输入命令：`go tool pprof -text democpu.pprof` ，输出：
![](https://img2020.cnblogs.com/blog/650581/202103/650581-20210330153439747-339748791.png)


#### 可视化分析

##### A. pprof 图形可视化

除了上面的命令行交互分析，还可以用图形化来分析程序性能。

图形化分析前，先要安装 graphviz 软件，
- 下载地址：[graphviz地址](https://graphviz.org/download/)，

下载对应的平台安装包，安装完成后，把执行文件 bin 放入 Path 环境变量中，然后在终端输入 `dot  -version`  命令查看是否安装成功。

**生成可视化文件：**

有 2 个步骤，根据上面采集的数据文件 democpu.pprof 来进行可视化：
1. 命令行输入：go tool pprof democpu.pprof
2. 输入 web 命令

在命令行里输入 web 命令，就可以生成一个 svg 格式的文件，用浏览器打开即可查看 svg 文件。

执行上面 2 个命令如下图：
![](https://img2020.cnblogs.com/blog/650581/202103/650581-20210329122233492-941911310.png)


用浏览器查看生成的 svg 图：

![](https://img2020.cnblogs.com/blog/650581/202103/650581-20210329122243497-1225791807.png)


(文件太大，只截取了一小部分图，完整的图请自行生成查看)

**关于图形的一点说明：**

1. 每个框代表一个函数，理论上框越大表示占用的 cpu 资源越多
2. 每个框之间的线条代表函数之间的调用关系，线条上的数字表示函数调用的次数
3. 每个框中第一行数字表示当前函数占用 cpu 的百分比，第二行数字表示当前函数累计占用 cpu 的百分比

##### B. web可视化-浏览器上查看数据

运行命令：`go tool pprof -http=:8080 democpu.pprof`

```sh
$ go tool pprof -http=:8080 democpu.pprof
Serving web UI on http://localhost:8080
```

命令运行完成后，会自动在浏览器上打开地址： `http://localhost:8080/ui/`，我们可以在浏览器上查看分析数据：
![](https://img2020.cnblogs.com/blog/650581/202103/650581-20210330041127682-216491175.png)
这张图就是上面用 web 命令生成的图。

> 如果你在 web 浏览时没有这么多菜单可供选择，那么请安装原生的 pprof 工具：
`go get -u github.com/google/pprof` ，然后在启动  `go tool pprof -http=:8080 democpu.pprof` ，就会出来菜单。

还可以查看火焰图， http 地址：http://localhost:8080/ui/flamegraph，可直接点击 VIEW 菜单下的 Flame Graph 选项查看火焰图。当然还有其他选项可供选择，比如 Top，Graph 等等选项。你可以根据需要选择。
![](https://img2020.cnblogs.com/blog/650581/202103/650581-20210330041153228-1227677383.png)

##### C. 火焰图 Flame Graph

其实上面的 web 可视化已经包含了火焰图，把火焰图集成到了 pprof 里。但为了向性能优化专家 Bredan Gregg 致敬，还是来体会一下火焰图生成过程。

火焰图 ([Flame Graph](https://github.com/brendangregg/FlameGraph)) 是性能优化专家 Bredan Gregg 创建的一种性能分析图。Flame Graphs visualize profiled code。

火焰图形状如下：
![](https://img2020.cnblogs.com/blog/650581/202103/650581-20210329122301550-1658411625.png)

（来自：https://github.com/brendangregg/FlameGraph）

上面用 pprof 生成的采样数据，要把它转换成火焰图，就要使用一个转换工具 [go-torch](github.com/uber/go-torch)，这个工具是 uber 开源，它是用 go 语言编写的，可以直接读取 pprof 采集的数据，并生成一张火焰图， svg 格式的文件。

1. 安装 go-torch：
> go get -v github.com/uber/go-torch

2. 安装 flame graph：
> git clone https://github.com/brendangregg/FlameGraph.git

并把 FlameGraph 安装目录位置添加进 Path 中。

3. 安装 perl 环境：

生成火焰图的程序 FlameGraph 是用 perl 写的，所以先要安装执行 perl 语言的环境。

- 安装 perl 环境：[https://www.perl.org/get.html](https://www.perl.org/get.html)
- 把执行文件 bin 加入 Path 中
- 在终端下执行命令：`perl -h` ，输出了帮助信息，则说明安装成功

![](https://img2020.cnblogs.com/blog/650581/202103/650581-20210329122317270-658836005.png)


4. 验证 FlameGraph 是否安装成功：

进入到 FlameGraph 安装目录，执行命令，`./flamegraph.pl --help`

![](https://img2020.cnblogs.com/blog/650581/202103/650581-20210329122328508-547114025.png)


输出信息说明安装成功

5. 生成火焰图：

重新进入到文件 democpu.pprof 的目录，然后执行命令：
> go-torch -b democpu.pprof

上面命令默认生成名为 torch.svg 的文件，用浏览器打开查看：
![](https://img2020.cnblogs.com/blog/650581/202103/650581-20210329122340804-1272269398.png)

自定义输出文件名，后面加 `-f` 参数：
> go-torch -b democpu.pprof -f cpu_flamegraph.svg

![](https://img2020.cnblogs.com/blog/650581/202103/650581-20210329122352322-482553191.png)


火焰图说明：

>火焰图 svg 文件，你可以点击上面的每个方块来查看分析它上面的内容。
>
>火焰图的调用顺序从下到上，每个方块代表一个函数，它上面一层表示这个函数会调用哪些函数，方块的大小代表了占用 CPU 使用时长长短。

go-torch 的命令格式：
```sh
go-torch [options] [binary] <profile source>
```

go-torch 帮助文档：
> 想了解更多 go-torch 用法，请用 help 命令查看帮助文档，`go-torch --help`。
> 
> 或查看 [go-torch README](https://github.com/uber-archive/go-torch/blob/master/README.md) 文档 。

> 我的公号：[九卷技术录：golang 性能优化分析工具 pprof (上) - 基础使用介绍](https://mp.weixin.qq.com/s/idHWKMyDWKHg3OdmvSIFvg) ，欢迎大家关注留言讨论

<br />
pprof 下一篇：

[golang 性能优化分析工具 pprof (下) - web 服务分析](https://www.cnblogs.com/jiujuan/p/14598141.html)

## 四、参考

- [pprof](https://github.com/google/pprof)
  - [README](https://github.com/google/pprof/blob/master/doc/README.md)
- [Profiling Go Programs](https://blog.golang.org/pprof)
- [runtime/pprof](https://golang.org/pkg/runtime/pprof/)
- [net/http/pprof](https://golang.org/pkg/net/http/pprof/)
- [go-torch](https://github.com/uber-archive/go-torch)
- [Flame Graph](https://github.com/brendangregg/FlameGraph)
