 **Linux amd64 + Go 1.19** 下，从 ELF 入口开始，对照源码，详细中文注释的“从 ELF 入口到 main.main”分析。

> 说明：代码片段都来自 Go 1.19 官方源码，关键位置给出文件路径和行号，你可以对照阅读。汇编基于 amd64 Go ABI。

## 0. 整体流程一览-全局视图

```mermaid
flowchart LR
  A[Linux kernel execve] --> B[_rt0_amd64_linux in rt0_linux_amd64.s]
  B --> C[_rt0_amd64 in asm_amd64.s]
  C --> D[runtime.rt0_go in asm_amd64.s]
  D --> E[runtime.args and runtime.osinit]
  E --> F[runtime.schedinit]
  F --> G[runtime.newproc - create main goroutine]
  G --> H[runtime.mstart - start M scheduler loop]
  H --> I[runtime.main in proc.go]
  I --> J[doInit - run all init]
  J --> K[main_main - user main.main]
```

下面按这个顺序，逐层展开。

## 1. Linux execve 如何进入 Go 的入口

**ELF 入口符号是 `_rt0_amd64_linux`**，定义在文件 `src/runtime/rt0_linux_amd64.s`  中，关键汇编如下：

```asm
// src/runtime/rt0_linux_amd64.s

#include "textflag.h"
// 普通可执行文件（-buildmode=exe）的入口
TEXT _rt0_amd64_linux(SB),NOSPLIT,$-8
    JMP    _rt0_amd64(SB)

// c-archive / c-shared 的入口（库模式）
TEXT _rt0_amd64_linux_lib(SB),NOSPLIT,$0
    JMP    _rt0_amd64_lib(SB)
```

逐行解释：

- `TEXT _rt0_amd64_linux(SB),NOSPLIT,$-8`
  - 定义一个函数 `_rt0_amd64_linux`，这是链接器看到的 ELF 入口符号。
  - Linux 加载器加载完 ELF 后，会将控制流跳转到这个地址。
  - `NOSPLIT`：告诉编译器这个函数不需要检查栈分裂（split stack），因为此时还没有 Go 栈。
  - `$-8`：表示 caller 栈帧大小为 0，但 8 字节对齐（细节可忽略）。
- `JMP _rt0_amd64(SB)`
  - 直接跳转到更通用的 amd64 启动函数 `_rt0_amd64`。
  - 不同 OS 的 `rt0_<os>_<arch>.s` 基本都跳到同一个 `_rt0_amd64`。


这一层的作用：只做平台适配，提供 Linux 特定的入口符号，真正工作交给 `_rt0_amd64`。


## 2. `_rt0_amd64`：到 argc/argv

`_rt0_amd64` 定义在文件 `src/runtime/asm_amd64.s` 中，关键汇编：

```asm
// src/runtime/asm_amd64.s
TEXT _rt0_amd64(SB),NOSPLIT,$-8
    MOVQ    0(SP), DI    // argc
    LEAQ    8(SP), SI    // argv
    JMP    runtime·rt0_go(SB)
```
逐行解释：

- Linux 在 execve 时，会把 `argc` 和 `argv` 放在主线程栈顶，布局类似：
  ```text
  high addr
  +-----------------+
  | argv[argc] = 0  |
  +-----------------+
  | argv[argc-1]    |
  +-----------------+
  | ...             |
  +-----------------+
  | argv[0]         |
  +-----------------+
  | argc            |
  +-----------------+ <--- SP
  ```
- `MOVQ 0(SP), DI`
  - 栈顶 0(SP) 就是 `argc`，把它放入 `DI` 寄存器。
  - Go 的入口函数约定通过 DI/SI 传参。
- `LEAQ 8(SP), SI`
  - 8(SP) 就是第一个 `argv[0]` 的地址。
  - LEAQ 把这个地址放入 `SI`，于是 `(SI)` 指向 `argv[0]`。
- `JMP runtime·rt0_go(SB)`
  - 跳转到 Go 运行时的真正启动函数 `rt0_go`，此时：
    - `DI` = `argc`
    - `SI` = `argv`

这一层的作用：把 OS 传进来的 C 风格参数 `argc/argv` 准备好，然后进入 Go 运行时初始化。


## 3. `runtime·rt0_go`：Go 运行时的“主引导记录”

`rt0_go` 定义在：文件：`src/runtime/asm_amd64.s` 中，下面分段看，只保留主干。

### 3.1 参数整理 & g0 栈初始化
```asm
// src/runtime/asm_amd64.s
TEXT runtime·rt0_go(SB),NOSPLIT|TOPFRAME,$0
    // copy arguments forward on an even stack
    MOVQ    DI, AX        // argc
    MOVQ    SI, BX        // argv
    SUBQ    $(5*8), SP    // 3args 2auto
    ANDQ    $~15, SP
    MOVQ    AX, 24(SP)
    MOVQ    BX, 32(SP)
    // create istack out of the given (operating system) stack.
    // _cgo_init may update stackguard.
    MOVQ    $runtime·g0(SB), DI
    LEAQ    (-64*1024+104)(SP), BX
    MOVQ    BX, g_stackguard0(DI)
    MOVQ    BX, g_stackguard1(DI)
    MOVQ    BX, (g_stack+stack_lo)(DI)
    MOVQ    SP, (g_stack+stack_hi)(DI)
```

**逐行解释：**

- `MOVQ DI, AX` / `MOVQ SI, BX`
  - 把 `argc/argv` 暂存到 AX/BX。
- `SUBQ $(5*8), SP` / `ANDQ $~15, SP`
  - 在栈上开辟 5 个槽位，并 16 字节对齐。
  - 这是 Go 调用约定需要的参数区。
- `MOVQ AX, 24(SP)` / `MOVQ BX, 32(SP)`
  - 把 `argc/argv` 保存到栈上，后续 Go 函数会去取。
- `MOVQ $runtime·g0(SB), DI`
  - 取出全局的 `g0` 地址，这是主线程的“调度栈 g”。
- `LEAQ (-64*1024+104)(SP), BX`
  - 在当前 SP 下方预留大约 64KB 作为 g0 的栈：
    - `stack_lo` = BX
    - `stack_hi` = SP
- `MOVQ BX, g_stackguard0(DI)` 等
  - 初始化 g0 的 stackguard 和栈范围，用于后续栈溢出检查。

这一段做了什么：

把 OS 栈的一部分划给 `g0` 使用；把 `argc/argv` 保存到 Go 栈参数区。


### 3.2 CPU 信息 & cgo 初始化

```asm
    // find out information about the processor we're on
    MOVL    $0, AX
    CPUID
    CMPL    AX, $0
    JE      nocpuinfo
    CMPL    BX, $0x756E6547    // "Genu"
    JNE     notintel
    CMPL    DX, $0x49656E69    // "ineI"
    JNE     notintel
    CMPL    CX, $0x6C65746E    // "ntel"
    JNE     notintel
    MOVB    $1, runtime·isIntel(SB)
notintel:
    // Load EAX=1 cpuid flags
    MOVL    $1, AX
    CPUID
    MOVL    AX, runtime·processorVersionInfo(SB)
nocpuinfo:
    // if there is an _cgo_init, call it.
    MOVQ    _cgo_init(SB), AX
    TESTQ   AX, AX
    JZ      needtls
    // arg 1: g0, already in DI
    MOVQ    $setg_gcc<>(SB), SI
    ... // 根据不同 OS 设置参数
    CALL    AX
```
**注释：**

- 通过 CPUID 检查是否 Intel CPU，记录到 `isIntel` 和 `processorVersionInfo`。
- 如果有 `_cgo_init`（cgo 程序会有），调用它：
  - 传参：`g0`、一个回调函数指针等，用来初始化 cgo 环境。


### 3.3 TLS / m0 / g0 绑定 & GOAMD64 检查

```asm
needtls:
    ...
    LEAQ    runtime·m0+m_tls(SB), DI
    CALL    runtime·settls(SB)
    // store through it, to make sure it works
    get_tls(BX)
    MOVQ    $0x123, g(BX)
    MOVQ    runtime·m0+m_tls(SB), AX
    CMPQ    AX, $0x123
    JEQ     2(PC)
    CALL    runtime·abort(SB)
ok:
    // set the per-goroutine and per-mach "registers"
    get_tls(BX)
    LEAQ    runtime·g0(SB), CX
    MOVQ    CX, g(BX)
    LEAQ    runtime·m0(SB), AX
    // save m->g0 = g0
    MOVQ    CX, m_g0(AX)
    // save m0 to g0->m
    MOVQ    AX, g_m(CX)
    CLD
```
**注释：**
- `settls`：设置线程本地存储（TLS），把 `m0.tls` 作为 TLS 槽位。
- 写入一个魔术值 0x123 再读回来验证 TLS 是否可用。
- 把 `g0` 和 `m0` 互相绑定：
  - `m0.g0 = g0`
  - `g0.m = m0`
- 这样每个 M（线程）就能通过 TLS 找到自己的 g。


### 3.4 调用 runtime·check / args / osinit / schedinit

```asm
    CALL    runtime·check(SB)
    MOVL    24(SP), AX    // copy argc
    MOVQ    AX, 0(SP)
    MOVQ    32(SP), AX    // copy argv
    MOVQ    AX, 8(SP)
    CALL    runtime·args(SB)
    CALL    runtime·osinit(SB)
    CALL    runtime·schedinit(SB)
```
**逐个函数作用：**
1. `runtime·check(SB)`做一些基本类型大小、对齐等的运行时检查，确保编译器假设成立。
2. `runtime·args(SB)`定义在 `runtime/runtime1.go`，简化版本类似：
   
   ```go
   var (
       argc int32
       argv **byte
   )
   func args(c int32, v **byte) {
       argc = c
       argv = v
       sysargs(c, v)
   }
   ```
   
   把刚才栈上的 `argc/argv` 保存到全局变量，供 `os.Args` 等使用。
   
3. `runtime·osinit(SB)`定义在 `runtime/os_linux.go`：
   
   ```go
   func osinit() {
       numCPUStartup = getCPUCount()
       physHugePageSize = getHugePageSize()
       ...
   }
   ```
   
   主要获取 CPU 核心数、页大小等 OS 级参数。
4. `runtime·schedinit(SB)`定义在 `runtime/proc.go`：
   
   ```go
   // The bootstrap sequence is:
   //
   //  call osinit
   //  call schedinit
   //  make & queue new G
   //  call runtime·mstart
   //
   // The new G calls runtime·main.
   func schedinit() {
       lockInit(&sched.lock, lockRankSched)
       ...
       stackinit()      // 栈管理初始化
       mallocinit()     // 内存分配器初始化
       cpuinit()        // CPU 特性相关
       ...
       // 创建 P 等
       procresize(getproccount())
   }
   ```

这是 **整个 GMP 调度器和内存管理的初始化核心**。


### 3.5 创建主 goroutine：`runtime·mainPC` + `newproc`

```asm
    // create a new goroutine to start program
    MOVQ    $runtime·mainPC(SB), AX    // entry
    PUSHQ    AX
    CALL    runtime·newproc(SB)
    POPQ    AX
```

**关键点：**

- `runtime·mainPC` 是一个指向 `runtime.main` 的函数值：
  ```asm
  // src/runtime/asm_amd64.s
  // mainPC is a function value for runtime.main, to be passed to newproc.
  // The reference to runtime.main is made via ABIInternal, since the
  // actual function (not the ABI0 wrapper) is needed by newproc.
  DATA runtime·mainPC+0(SB)/8,$runtime·main<ABIInternal>(SB)
  GLOBL runtime·mainPC(SB),RODATA,$8
  ```
- `newproc` 的语义：
  - `newproc(fn)` 会创建一个新的 goroutine，入口函数是 `fn`。
  - 在这里，`fn` 就是 `runtime.main`，所以这是 **主 goroutine（main goroutine）的创建**。

**此时：**

- 还在主线程（M0）上，g0 是调度栈；
- 新创建的 G 就是 main goroutine，入口是 `runtime.main`。


### 3.6 启动调度循环：`runtime·mstart`

```asm
    // start this M
    CALL    runtime·mstart(SB)
    CALL    runtime·abort(SB)    // mstart should never return
    RET
```
- `runtime·mstart` 定义也在 `asm_amd64.s`：
  ```asm
  TEXT runtime·mstart(SB),NOSPLIT|TOPFRAME,$0
      CALL    runtime·mstart0(SB)
      RET
  ```
- `mstart0` 定义在 `runtime/proc.go`，核心是进入调度循环：
  - 当前 M 开始执行调度器 `schedule()`；
  - 从全局或本地队列中挑选可运行的 G；
  - 通过 `gogo` 跳到 G 的栈执行。

从现在开始：

- 主 M 进入调度循环；
- 最终会调度到刚才创建的 main goroutine，执行 `runtime.main`。


## 4. `runtime.main`：从 runtime 到用户 `main.main`

`runtime.main` 定义在文件：`src/runtime/proc.go` ，关键部分（简化）：

```go
// src/runtime/proc.go
//go:linkname main_main main.main
func main_main()
// The main goroutine.
func main() {
    g := getg()
    // Racectx of m0->g0 is used only as the parent of the main goroutine.
    g.m.g0.racectx = 0
    // Max stack size is 1 GB on 64-bit, 250 MB on 32-bit.
    if goarch.PtrSize == 8 {
        maxstacksize = 1000000000
    } else {
        maxstacksize = 250000000
    }
    maxstackceiling = 2 * maxstacksize
    // Allow newproc to start new Ms.
    mainStarted = true
    if GOARCH != "wasm" {
        systemstack(func() {
            newm(sysmon, nil, -1)
        })
    }
    lockOSThread()
    if g.m != &m0 {
        throw("runtime.main not on m0")
    }
    runtimeInitTime = nanotime()
    // 执行 runtime 包自身的 init
    doInit(&runtime_inittask)
    gcenable()
    main_init_done = make(chan bool)
    if iscgo {
        // cgo 相关初始化
    }
    // 执行用户包的 init
    doInit(&main_inittask)
    close(main_init_done)
    needUnlock = false
    unlockOSThread()
    if isarchive || islibrary {
        // c-archive / c-shared 模式，不调用 main.main
        return
    }
    fn := main_main   // main_main 链接到用户 main.main
    fn()              // 调用用户的 main.main
    // 处理可能的 panic 等情况后退出
    exit(0)
}
```

**逐段解释：**
1. `//go:linkname main_main main.main`
   - 通过 linkname，把 `main_main` 绑定到用户包中的 `main.main`。
   - 这样 runtime 就能通过 `main_main()` 调用你的 main 函数。
2. `mainStarted = true`标记主 M 已经启动，后续 `newproc` 等可以使用。
3. `newm(sysmon, ...)`
   - 启动后台监控线程 `sysmon`，负责：
     - 抢占长时间运行的 G；
     - 触发强制 GC；
     - 释放闲置 P 等。
4. `lockOSThread()`
   - 在 init 阶段，将 main goroutine 锁在主 OS 线程上。
   - 如果你需要在 main 中调用必须由主线程执行的 C 库，可以在 init 里调用 `runtime.LockOSThread()` 保持这个锁。
5. `doInit(&runtime_inittask)`执行 runtime 包自身的 init 任务。
6. `gcenable()`启用 GC。
7. `doInit(&main_inittask)`执行用户包的 init 函数（依赖排序已经在编译时完成）。
8. `fn := main_main; fn()`通过 `main_main` 调用用户写的 `main.main`。

**从这里开始，我们写的代码才真正开始执行。**




## 5. 总结：从 ELF 入口到 main.main 的完整链路

把前面几层串起来，就是：
1. Linux kernel execve
   - 加载 ELF，入口指向 `_rt0_amd64_linux`（在 `rt0_linux_amd64.s` 中）。
2. `_rt0_amd64_linux`
   - 跳转到 `_rt0_amd64`（`asm_amd64.s`）。
3. `_rt0_amd64`
   - 从栈上取出 `argc/argv`，放入 DI/SI；
   - 跳转到 `runtime·rt0_go`。
4. `runtime·rt0_go`
   - 初始化 g0 栈、TLS、m0/g0；
   - 调用：
     - `runtime.check`
     - `runtime.args`：保存 `argc/argv`
     - `runtime.osinit`：获取 CPU 核数等
     - `runtime.schedinit`：初始化调度器、内存分配器、GC、P 等
   - 通过 `newproc(runtime.mainPC)` 创建 main goroutine；
   - 调用 `runtime.mstart`，让当前 M 进入调度循环。
5. `runtime.main`（在 main goroutine 中执行）
   - 设置最大栈大小；
   - 启动 `sysmon`；
   - 执行 runtime / 用户 init；
   - 通过 `main_main` 调用用户的 `main.main`；
   - `main.main` 返回后，调用 `exit(0)` 退出进程。

## 参考

- https://golang.design/go-questions/sched/init/ Go scheduler初始化 《Go 程序员面试笔试宝典》 *作者*: 饶全成, 欧长坤, 楚秦 等编著
- https://golang.design/go-questions/compile/link-process/ Go编译链接过程 《Go 程序员面试笔试宝典》 *作者*: 饶全成, 欧长坤, 楚秦 等编著
- https://golang.design/go-questions/compile/booting/ Go程序启动过程 《Go 程序员面试笔试宝典》 
- https://golang.design/go-questions/compile/cmd/ Go编译相关的命令
- https://github.com/golang/go/blob/release-branch.go1.19/src/runtime/proc.go go proc.go
- https://www.cnblogs.com/jiujuan/p/16555192.html Go汇编ASM基础学习
- https://github.com/yifengyou/parser-elf  ELF解释器及相关学习笔记
- https://cloud.tencent.com/developer/article/2187999  深入了解 Go ELF 信息
