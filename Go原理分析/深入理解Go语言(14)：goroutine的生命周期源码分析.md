Go编程语言在 goroutine 生命周期简析

> 基于 go1.19

## GPM全局图

在前面文章中，有一张 GPM 的全局运行示意图，起到总揽全局的作用：

```ascii
                            +-------------------- sysmon ---------------//------+ 
                            |                                                   |
                            |                                                   |
               +---+      +---+-------+                   +--------+          +---+---+
go func() ---> | G | ---> | P | local | <=== balance ===> | global | <--//--- | P | M |
               +---+      +---+-------+                   +--------+          +---+---+
                            |                                 |                 | 
                            |      +---+                      |                 |
                            +----> | M | <--- findrunnable ---+--- steal <--//--+
                                   +---+ 
                                     |
                                   mstart
                                     |
              +--- execute <----- schedule 
              |                      |   
              |                      |
              +--> G.fn --> goexit --+ 


              1. go func() 语气创建G。
              2. 将G放入P的本地队列（或者平衡到全局全局队列）。
              3. 唤醒或新建M来执行任务。
              4. 进入调度循环
              5. 尽力获取可执行的G，并执行
              6. 清理现场并且重新进入调度循环

```



## 一、核心数据结构定义

### 1.1 Goroutine 结构体 (`g`)

**源码位置**: `src/runtime/runtime2.go`

```go
// src/runtime/runtime2.go (Go 1.19)
type g struct {
    // 栈信息
    stack       stack   // 栈的起止地址 [lo, hi)
    stackguard0 uintptr // 栈溢出检查点（用于抢占式调度）
    stackguard1 uintptr // 用于 stack 增长检查
    
    // 调度上下文（保存寄存器状态）
    sched       gobuf   // 保存的寄存器上下文（SP, PC等）
    
    // 状态与标识
    status      uint32  // G的状态 (_Gidle, _Grunnable, _Grunning, _Gwaiting, _Gdead...)
    atomicstatus uint32 // 原子操作的状态副本
    
    // 关联关系
    m           *m      // 当前绑定的M（Machine）
    p           *p      // 当前关联的P（Processor）
    
    // 函数调用信息
    gopc        uintptr // 创建此G的go语句的PC
    startpc     uintptr // G的函数入口PC
    waitreason  string  // 等待原因（调试用）
    
    // 其他重要字段
    goid        int64   // Goroutine ID（唯一标识）
    preempt     bool    // 抢占标志
    panicking   bool    // 是否正在panic
    locks       int32   // 锁计数（>0表示不允许抢占）
    
    // 阻塞相关
    waitlock    unsafe.Pointer // 等待的锁
    waittraceev byte    // 跟踪事件
    waittraceskip int   // 跟踪跳过
    
    // 系统调用相关
    syscallsp   uintptr // 系统调用时的SP
    syscallpc   uintptr // 系统调用时的PC
    
    // 更多字段... (Go 1.19中g结构体约有70+字段)
}
```

### 1.2 Machine 结构体 (`m`)

**源码位置**: `src/runtime/runtime2.go`

```go
// src/runtime/runtime2.go (Go 1.19)
type m struct {
    g0      *g     // 执行调度器的特殊G（系统栈）
    mstartfn func()
    curg    *g     // 当前正在执行的用户G
    p       puintptr // 绑定的P（原子操作）
    nextp   puintptr // 下一个要绑定的P
    id      int32  // M的ID
    spinning bool  // 是否正在寻找工作
    blocked   bool  // 是否被阻塞
    
    // 系统调用相关
    locks     int32  // 锁计数
    gcing     bool   // 是否正在GC
    park      note   // 用于park的note
    
    // 缓存
    pcache    puintptr // P缓存
    
    // 更多字段...
}
```

### 1.3 Processor 结构体 (`p`)

**源码位置**: `src/runtime/runtime2.go`

```go
// src/runtime/runtime2.go (Go 1.19)
type p struct {
    lock    mutex
    id      int32
    status  uint32  // _Pidle, _Prunning, _Psyscall, _Pgcstop, _Pdead
    
    // 运行队列
    runqhead uint32  // 队列头索引
    runqtail uint32  // 队列尾索引
    runq     [256]guintptr // 本地运行队列（最多256个G）
    
    // 全局队列缓存
    gfree   *g      // 空闲G链表
    gfreecnt int32  // 空闲G数量
    
    // 调度统计
    schedtick   uint32  // 每次调度调用递增
    syscalltick uint32  // 每次系统调用递增
    
    // 关联的M
    m           muintptr // 当前绑定的M
    
    // 更多字段...
}
```

### 1.4 调度上下文 (`gobuf`)

```go
// src/runtime/runtime2.go (Go 1.19)
type gobuf struct {
    sp   uintptr  // 栈指针
    pc   uintptr  // 程序计数器
    g    guintptr // 指向G的指针
    ctxt unsafe.Pointer
    ret  uintptr
    lr   uintptr
    bp   uintptr  // 帧指针（frame pointer）
}
```



## 二、Goroutine 状态定义

**源码位置**: `src/runtime/runtime2.go`

```go
// src/runtime/runtime2.go (Go 1.19)
const (
    _Gidle = iota // 0: 刚分配，尚未初始化
    _Grunnable    // 1: 在运行队列中，等待执行
    _Grunning     // 2: 正在执行用户代码
    _Gsyscall     // 3: 正在执行系统调用
    _Gwaiting     // 4: 被阻塞，等待某事件
    _Gdead        // 6: 已死亡，等待回收
    _Gcopystack   // 7: 栈正在被拷贝（GC或栈增长）
)
```


## 三、Goroutine 生命周期完整流程图

```ascii
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Goroutine 生命周期完整流程图                          │
└─────────────────────────────────────────────────────────────────────────────┘

                              ┌──────────────┐
                              │  程序启动     │
                              │ main.main()  │
                              └──────┬───────┘
                                     │
                                     ▼
                    ┌────────────────────────────────┐
                    │      1. 创建 Goroutine          │
                    │      go func() { ... }         │
                    │      runtime.newproc()         │
                    └───────────────┬────────────────┘
                                    │
                                    ▼
                    ┌────────────────────────────────┐
                    │      2. 初始化 G 结构体          │
                    │      runtime.newproc1()        │
                    │      - 分配 g 结构体             │
                    │      - 初始化栈 (2KB)           │
                    │      - 设置入口为 goexit        │
                    │      - 状态: _Gidle → _Grunnable│
                    └───────────────┬────────────────┘
                                    │
                                    ▼
                    ┌────────────────────────────────┐
                    │      3. 放入运行队列            │
                    │      runqput()                 │
                    │      - P 本地队列 (优先)        │
                    │      - 全局队列 (队列满时)      │
                    └───────────────┬────────────────┘
                                    │
                    ┌───────────────┴───────────────┐
                    │                               │
                    ▼                               │
        ┌───────────────────────┐                   │
        │   4. 调度器选取 G      │                   │
        │   runtime.schedule()  │                   │
        │   - runqget() 本地队列 │                   │
        │   - globrunqget() 全局 │                   │
        │   - stealWork() 偷取   │                   │
        └───────────┬───────────┘                   │
                    │                               │
                    ▼                               │
        ┌───────────────────────┐                   │
        │   5. 执行 G            │                   │
        │   runtime.execute()   │                   │
        │   状态: _Grunnable →  │                   │
        │         _Grunning     │                   │
        │   绑定 M & P          │                   │
        │   gogo(&g.sched)      │                   │
        └───────────┬───────────┘                   │
                    │                               │
        ┌───────────┼───────────┐                   │
        │           │           │                   │
        ▼           ▼           ▼                   │
┌───────────┐ ┌───────────┐ ┌───────────┐           │
│ 需要阻塞   │ │ 执行完毕   │ │ 系统调用   │           │
│ gopark()  │ │ goexit()  │ │ entersyscall│          │
└─────┬─────┘ └─────┬─────┘ └─────┬─────┘           │
      │             │             │                   │
      ▼             ▼             ▼                   │
┌───────────┐ ┌───────────┐ ┌───────────┐           │
│ 6. 阻塞   │ │ 7. 死亡   │ │ _Gsyscall │           │
│ _Gwaiting │ │ _Gdead    │ │           │           │
│ 保存上下文 │ │ 清理栈    │ │ 可能释放 P │           │
│ 解绑 M    │ │ 回收入池  │ │           │           │
└─────┬─────┘ └─────┬─────┘ └─────┬─────┘           │
      │             │             │                   │
      │             │             └────────┐          │
      │             │                      │          │
      ▼             │                      ▼          │
┌───────────┐       │            ┌─────────────────┐  │
│ 等待事件   │       │            │ 系统调用返回     │  │
│ (channel/ │       │            │ exitsyscall()   │  │
│  lock/    │       │            │ 状态: _Grunning │  │
│  sleep)   │       │            └────────┬────────┘  │
└─────┬─────┘       │                     │          │
      │             │                     │          │
      │  事件就绪    │                     │          │
      │  (唤醒)     │                     │          │
      ▼             │                     │          │
┌───────────┐       │                     │          │
│ 8. 唤醒   │       │                     │          │
│ goready() │       │                     │          │
│ _Gwaiting │       │                     │          │
│   →       │       │                     │          │
│ _Grunnable│       │                     │          │
│ 放入队列   │       │                     │          │
└─────┬─────┘       │                     │          │
      │             │                     │          │
      └─────────────┴─────────────────────┘          │
                    │                                │
                    └────────────────────────────────┘
                              │
                              ▼
                    ┌───────────────────────┐
                    │   返回步骤 4 继续调度   │
                    │   (循环直到所有 G 完成) │
                    └───────────────────────┘
```



## 四、核心代码流程分析 

### 4.1 创建 Goroutine (`newproc` → `newproc1`)

**源码位置**: `src/runtime/proc.go`

```go
// src/runtime/proc.go (Go 1.19)
// 编译器将 go 语句转换为此函数调用
func newproc(fn *funcval) {
    // 获取当前正在运行的 G
    gp := getg()
    // 获取调用者的 PC（用于调试和追踪）
    pc := sys.GetCallerPC()
    
    // 在系统栈上执行（避免用户栈溢出）
    systemstack(func() {
        // 创建新的 G
        newg := newproc1(fn, gp, pc, false, waitReasonZero)
        
        // 获取当前 P
        pp := getg().m.p.ptr()
        
        // 将新 G 放入 P 的运行队列
        // 第三个参数 true 表示尝试唤醒空闲的 P/M
        runqput(pp, newg, true)
    })
}

// newproc1 - 实际创建 G 的核心函数
func newproc1(fn *funcval, callergp *g, callerpc uintptr, parentFrame unsafe.Pointer, waitreason waitReason) *g {
    // 获取当前 M 和 P
    _g_ := getg()
    mp := _g_.m
    
    // 从 P 的空闲 G 缓存池分配 g 结构体
    newg := gfget(_g_.m.p.ptr())
    if newg == nil {
        // 缓存池为空，分配新的 g 结构体
        newg = malg(_StackMin) // _StackMin = 2KB (初始栈大小)
        newg.sched.sp = newg.stack.lo + _StackMin
    }
    
    // 初始化新 G 的各个字段
    newg.sched.pc = abi.FuncPCABI0(goexit) + sys.PCQuantum // 设置返回地址为 goexit
    newg.startpc = fn.fn                                   // 用户函数的入口
    newg.sched.g = guintptr(unsafe.Pointer(newg))
    newg.goid = int64(atomic.Xadd64(&sched.goidgen, 1))    // 分配唯一的 G ID
    newg._panic = nil
    newg.openDefer = nil
    newg.sched.sp = newg.stack.lo + _StackMin
    newg.stackguard0 = newg.stack.lo + _StackMin
    newg.waitreason = ""
    newg.preempt = true
    newg.gopc = callerpc
    newg.ancestors = saveAncestors(callergp)
    newg.startpc = fn.fn
    
    // 设置新 G 的父 G 信息（用于追踪）
    if isSystemGoroutine(newg) {
        atomic.Xadd(&sched.ngsys, +1)
    }
    newg.labels = nil
    newg.cgoCtxt = nil
    
    // 保存调用者的上下文（用于调试）
    if trace.enabled {
        traceGoCreate()
    }
    
    // 初始化新 G 的栈帧，将用户函数参数压入栈
    gftrace(newg)
    gogo(&newg.sched) // 这个调用实际上不会返回，只是设置上下文
    
    return newg
}
```

### 4.2 调度器核心 (`schedule`)

**源码位置**: `src/runtime/proc.go`

```go
// src/runtime/proc.go (Go 1.19)
// schedule - 调度器主循环，寻找可运行的 G 并执行
func schedule() {
    _g_ := getg()
    
    // 1. GC 相关工作（如果需要）
    if gp == nil {
        if _g_.m.p.ptr().gcBgMarkWorker != nil {
            // 处理 GC 后台标记工作
        }
        
        // 全局调度检查
        if sched.gcwaiting != 0 {
            gcstopm() // GC 等待时停止 M
        }
    }
    
    // 2. 寻找可运行的 G
    var gp *g
    var inheritTime bool
    
    if gp == nil {
        // 2.1 检查本地运行队列（50% 概率优先检查，避免饥饿）
        if _g_.m.p.ptr().schedtick%2 == 0 && sched.runqsize != 0 {
            lock(&sched.lock)
            gp = globrunqget(_g_.m.p.ptr(), 1)
            unlock(&sched.lock)
        }
        // 2.2 从 P 的本地队列获取
        if gp == nil {
            gp, inheritTime = runqget(_g_.m.p.ptr())
        }
        // 2.3 如果本地队列为空，尝试从全局队列获取
        if gp == nil {
            gp = globrunqget(_g_.m.p.ptr(), 0)
        }
    }
    
    // 3. 如果还是没有 G，尝试从其他 P 偷取工作（Work Stealing）
    if gp == nil {
        gp, inheritTime = findrunnable() // blocks until work is available
    }
    
    // 4. 执行找到的 G
    if gp == nil {
        throw("schedule: gp is nil")
    }
    
    // 5. 执行 G
    execute(gp, inheritTime)
}

// execute - 实际执行 G
func execute(gp *g, inheritTime bool) {
    _g_ := getg()
    
    // 设置 G 的状态为 _Grunning
    casgstatus(gp, _Grunnable, _Grunning)
    
    // 绑定 G 到当前 M
    gp.waitsince = 0
    gp.preempt = false
    gp.stackguard0 = gp.stack.lo + _StackMin
    
    // 将 G 关联到当前 M
    _g_.m.curg = gp
    gp.m = _g_.m
    
    // 6. 上下文切换，跳转到 G 的代码执行
    // gogo 是汇编函数，恢复 G 的寄存器状态并开始执行
    gogo(&gp.sched)
}
```

### 4.3 阻塞与挂起 (`gopark`)

**源码位置**: `src/runtime/proc.go`

```go
// src/runtime/proc.go (Go 1.19)
// gopark - 阻塞当前 G，直到被唤醒
// 用于 channel 操作、锁、sleep 等阻塞场景
func gopark(unlockf func(*g, unsafe.Pointer) bool, lock unsafe.Pointer, reason string, traceEv byte, traceskip int) {
    mp := getg().m
    gp := mp.curg
    
    // 1. 修改 G 的状态为 _Gwaiting
    status := readgstatus(gp)
    if status != _Grunning {
        throw("gopark: bad g status")
    }
    
    // 2. 保存等待信息（用于调试和追踪）
    mp.waitlock = lock
    mp.waittraceev = traceEv
    mp.waittraceskip = traceskip
    gp.waitreason = reason
    
    // 3. 调用 unlockf 释放锁（如果提供了）
    // 这允许其他 G 继续执行
    if unlockf != nil && !unlockf(gp, lock) {
        // 如果解锁失败，特殊处理
        mp.waitlock = nil
        mp.waittraceev = 0
        gp.waitreason = ""
        return
    }
    
    // 4. 关键：保存当前 G 的上下文，并切换到调度器
    // mcall 会保存当前 G 的寄存器到 g.sched，然后切换到 g0 栈执行 park_m
    mcall(park_m)
    
    // 当 G 被唤醒后，从这里返回
    // 此时 G 的状态已经被改为 _Grunnable，并且上下文已恢复
    mp.waitlock = nil
    mp.waittraceev = 0
    gp.waitreason = ""
}

// park_m - 在 g0 栈上执行，实际执行 park 操作
func park_m(gp *g) {
    _g_ := getg()
    
    if trace.enabled {
        traceGoPark()
    }
    
    // 切换回调度器，寻找其他 G 执行
    schedule()
}
```

### 4.4 唤醒 G (`goready`)

**源码位置**: `src/runtime/proc.go`

```go
// src/runtime/proc.go (Go 1.19)
// goready - 唤醒一个被阻塞的 G
func goready(gp *g, traceskip int) {
    systemstack(func() {
        ready(gp, traceskip, true)
    })
}

// ready - 将 G 状态改为 _Grunnable 并放入运行队列
func ready(gp *g, traceskip int, next bool) {
    status := readgstatus(gp)
    
    // 1. 修改状态：_Gwaiting → _Grunnable
    if !casgstatus(gp, _Gwaiting, _Grunnable) {
        throw("ready: bad g status")
    }
    
    // 2. 追踪（如果启用）
    if trace.enabled {
        traceGoUnpark(gp, traceskip)
    }
    
    // 3. 将 G 放入运行队列
    pp := getg().m.p.ptr()
    runqput(pp, gp, next)
    
    // 4. 如果需要，唤醒空闲的 P/M
    if next {
        nextp := pp
        if !nextp.runnext.empty() {
            // 如果 P 的 runnext 有 G，可能需要唤醒其他 M
        }
        if sched.npidle != 0 && sched.nmspinning == 0 {
            wakep() // 唤醒一个空闲的 P
        }
    }
}

// runqput - 将 G 放入 P 的运行队列
func runqput(_p_ *p, gp *g, next bool) {
    if randomizeScheduler && next && fastrand()%2 == 0 {
        next = false
    }
    
    // 尝试放入 runnext（下一个优先执行的 G）
    if next {
    retryNext:
        oldnext := _p_.runnext
        if !_p_.runnext.cas(oldnext, guintptr(unsafe.Pointer(gp))) {
            goto retryNext
        }
        if oldnext == 0 {
            return
        }
        // 如果 runnext 原来有 G，把它放到普通队列
        gp = oldnext.ptr()
    }
    
    // 放入普通运行队列
retry:
    h := atomic.LoadAcq(&_p_.runqhead)
    t := _p_.runqtail
    if t-h < uint32(len(_p_.runq)) {
        _p_.runq[t%uint32(len(_p_.runq))] = guintptr(unsafe.Pointer(gp))
        atomic.StoreRel(&_p_.runqtail, t+1)
        return
    }
    
    // 队列满了，放入全局队列
    if runqputslow(_p_, gp, h, t) {
        return
    }
    goto retry
}
```

### 4.5 G 执行完毕 (`goexit`)

**源码位置**: `src/runtime/asm_amd64.s` 和 `src/runtime/proc.go`

```go
// src/runtime/asm_amd64.s (Go 1.19)
// 汇编实现的 goexit，每个 G 的栈底都设置这个返回地址
TEXT runtime·goexit(SB),NOSPLIT,$0-0
    CALL    runtime·goexit1(SB)  // 调用 Go 实现的清理函数
    RET

// src/runtime/proc.go (Go 1.19)
// goexit1 - Go 实现的 G 退出清理
func goexit1() {
    if raceenabled {
        racegoend()
    }
    if trace.enabled {
        traceGoEnd()
    }
    
    // 关键：切换到 g0 栈执行 goexit0
    mcall(goexit0)
}

// goexit0 - 在 g0 栈上执行，清理 G 并回收入池
func goexit0(gp *g) {
    _g_ := getg()
    
    // 1. 清理各种状态
    for {
        // 清理 panic 和 defer
        if gp._panic != nil {
            // 处理未处理的 panic
        }
        // 清理 defer
        if gp.openDefer != nil {
            // 清理未执行的 defer
        }
        break
    }
    
    // 2. 清理栈
    gp.stackguard0 = 0
    gp.stackguard1 = 0
    
    // 3. 修改状态：_Grunning → _Gdead
    casgstatus(gp, _Grunning, _Gdead)
    
    // 4. 清理关联
    gp.m = nil
    gp.p = nil
    _g_.m.curg = nil
    
    // 5. 将 G 放回 P 的空闲缓存池
    gp.sched.sp = 0
    gp.sched.pc = 0
    gp.sched.g = 0
    gp.sched.ctxt = nil
    gp.sched.bp = 0
    
    // 放入 gfree 链表
    gfput(_g_.m.p.ptr(), gp)
    
    // 6. 进入调度器，寻找下一个 G 执行
    schedule()
}
```


## 五、上下文切换机制 (`gogo` & `mcall`)

### 5.1 `mcall` - 切换到调度器

**源码位置**: `src/runtime/asm_amd64.s`

```assembly
// src/runtime/asm_amd64.s (Go 1.19)
// mcall(fn) - 从用户 G 切换到 g0（调度器 G）
// 输入：fn 是要在 g0 上执行的函数
TEXT runtime·mcall(SB),NOSPLIT,$0-2
    MOVQ    DI, R10     // 保存 fn 参数
    MOVQ    8(SP), R11  // 保存返回地址
    
    // 1. 保存当前 G 的上下文
    MOVQ    SP, 0(SP)   // 保存 SP
    MOVQ    R11, 8(SP)  // 保存 PC（返回地址）
    MOVQ    BP, 16(SP)  // 保存 BP
    
    // 2. 获取当前 G
    MOVQ    g(CX), BX   // BX = current g
    
    // 3. 切换到 g0 栈
    MOVQ    g_g0(BX), BX    // BX = g0
    MOVQ    g_stack_hi(BX), SP  // SP = g0.stack.hi
    
    // 4. 调用 fn（在 g0 栈上执行）
    MOVQ    R10, DI       // fn 作为第一个参数
    CALL    runtime·mcallfn(SB)
```

### 5.2 `gogo` - 恢复 G 执行

**源码位置**: `src/runtime/asm_amd64.s`

```assembly
// src/runtime/asm_amd64.s (Go 1.19)
// gogo(buf) - 从 gobuf 恢复 G 的上下文并执行
// 输入：buf 指向 gobuf 结构
TEXT runtime·gogo(SB),NOSPLIT,$0
    MOVQ    DI, BX        // BX = buf
    
    // 1. 从 gobuf 恢复寄存器
    MOVQ    gobuf_sp(BX), SP  // 恢复 SP
    MOVQ    gobuf_pc(BX), DX  // 获取 PC
    MOVQ    gobuf_g(BX), BX   // BX = g
    MOVQ    gobuf_bp(BX), BP  // 恢复 BP
    
    // 2. 更新当前 M 的 curg
    MOVQ    BX, g(CX)     // 更新 TLS 中的 g
    
    // 3. 跳转到 PC 继续执行
    JMP     DX            // 跳转到保存的 PC
```


## 六、GMP 调度架构总览

```ascii
┌────────────────────────────────────────────────────────────────────────────┐
│                           Go 1.19 GMP 调度架构                              │
└────────────────────────────────────────────────────────────────────────────┘

    用户代码层                    Runtime 调度层                  操作系统层
┌──────────────┐            ┌────────────────────┐            ┌──────────────┐
│              │            │                    │            │              │
│  Goroutine   │            │      Processor     │            │   Machine    │
│      G1      │◄──────────►│         P1         │◄──────────►│      M1      │◄─── OS Thread 1
│  (_Grunning) │  执行      │  (runq[256])       │  绑定      │  (curg=G1)   │
│              │            │                    │            │              │
└──────────────┘            └────────────────────┘            └──────────────┘
       ▲                              ▲                               ▲
       │                              │                               │
       │                              │                               │
┌──────────────┐            ┌────────────────────┐            ┌──────────────┐
│              │            │                    │            │              │
│  Goroutine   │            │      Processor     │            │   Machine    │
│      G2      │◄──────────►│         P2         │◄──────────►│      M2      │◄─── OS Thread 2
│  (_Grunnable)│  等待      │  (runq[256])       │  绑定      │  (curg=?)    │
│   [队列中]    │            │                    │            │              │
└──────────────┘            └────────────────────┘            └──────────────┘
       ▲                              ▲                               ▲
       │                              │                               │
       │                              │                               │
┌──────────────┐            ┌────────────────────┐            ┌──────────────┐
│              │            │                    │            │              │
│  Goroutine   │            │      Processor     │            │   Machine    │
│      G3      │◄──────────►│         P3         │◄──────────►│      M3      │◄─── OS Thread 3
│  (_Gwaiting) │  阻塞      │  (runq[256])       │  绑定      │  (curg=?)    │
│  [等待事件]   │            │                    │            │              │
└──────────────┘            └────────────────────┘            └──────────────┘
       │                              │                               │
       │                              │                               │
       ▼                              ▼                               ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Global Run Queue                                │
│                         (全局运行队列，有锁保护)                              │
└─────────────────────────────────────────────────────────────────────────────┘
       ▲
       │
       │
┌─────────────────────────────────────────────────────────────────────────────┐
│                              G Free List                                     │
│                         (空闲 G 缓存池，按 P 分布)                            │
└─────────────────────────────────────────────────────────────────────────────┘

关键关系:
├── G (Goroutine): 用户态协程，包含栈和调度上下文
├── M (Machine): 操作系统线程，真正执行代码的实体
└── P (Processor): 逻辑处理器，持有 G 的运行队列，M 需要绑定 P 才能执行 G
```

调度流程:
1. M 绑定 P
2. P 从本地 runq 获取 G
3. M 执行 G（G 状态: _Grunnable → _Grunning）
4. G 阻塞时：gopark → _Gwaiting，M 寻找其他 G
5. G 完成时：goexit → _Gdead，回收到 gfree
6. G 被唤醒时：goready → _Grunnable，放回 runq


## 七、关键调度时机总结

| 时机 | 触发条件 | 相关函数 | 状态变化 |
|------|----------|----------|----------|
| **创建** | `go func()` | `newproc` → `newproc1` | `_Gidle` → `_Grunnable` |
| **调度执行** | 调度器循环 | `schedule` → `execute` → `gogo` | `_Grunnable` → `_Grunning` |
| **主动让出** | `runtime.Gosched()` | `gosched_m` | `_Grunning` → `_Grunnable` |
| **阻塞等待** | channel/lock/sleep | `gopark` → `park_m` | `_Grunning` → `_Gwaiting` |
| **被唤醒** | 事件就绪 | `goready` → `ready` | `_Gwaiting` → `_Grunnable` |
| **系统调用** | syscall | `entersyscall` / `exitsyscall` | `_Grunning` ↔ `_Gsyscall` |
| **执行完毕** | 函数返回 | `goexit` → `goexit0` | `_Grunning` → `_Gdead` |
| **被抢占** | 栈检查/时间片 | `preempt` | `_Grunning` → `_Grunnable` |



## 参考

- https://github.com/13meimei/night-reading-go/blob/master/reading/20180802/README.md  golang中goroutine的调度
- https://golang.design/go-questions/sched/init/  scheduler 初始化
- https://chasecs.github.io/posts/go-scheduler-introduction/ Go 源码分析：scheduler 工作流程
- https://zboya.github.io/post/go_scheduler/  深入golang runtime的调度

