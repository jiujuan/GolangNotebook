在前面[一节中](https://www.cnblogs.com/jiujuan/p/12735559.html)简单介绍了golang的调度模型-GPM模型，介绍了他们各自的作用。这篇文章就来看看他们的源码结构。
> Go版本：go1.13.9

## M 结构体

M 结构体是 OS 线程的一个抽象，主要负责结合 P 运行 G，真正执行 goroutine，做系统调用。

它里面有很多字段，差不多有60个字段，我们看看里面主要的字段意思。
/src/runtime/runtime2.go

```go
type m struct {
    // 系统管理的一个g，执行调度代码时使用的。比如执行用户的goroutine时，就需要把把用户
    // 的栈信息换到内核线程的栈，以便能够执行用户goroutine
	g0      *g     // goroutine with scheduling stack
	morebuf gobuf  // gobuf arg to morestack
	divmod  uint32 // div/mod denominator for arm - known to liblink

	// Fields not known to debuggers.
	procid        uint64       // for debuggers, but offset not hard-coded
    //处理signal的 g
	gsignal       *g           // signal-handling g
	goSigStack    gsignalStack // Go-allocated signal handling stack
	sigmask       sigset       // storage for saved signal mask
    //线程的本地存储TLS，这里就是为什么OS线程能运行M关键地方
	tls           [6]uintptr   // thread-local storage (for x86 extern register)
	//go 关键字运行的函数
    mstartfn      func()
    //当前运行的用户goroutine的g结构体对象
	curg          *g       // current running goroutine
	caughtsig     guintptr // goroutine running during fatal signal
    
    //当前工作线程绑定的P，如果没有就为nil
	p             puintptr // attached p for executing go code (nil if not executing go code)
	//暂存与当前M潜在关联的P
    nextp         puintptr
    //M之前调用的P
	oldp          puintptr // the p that was attached before executing a syscall
	id            int64
	mallocing     int32
	throwing      int32
    //当前M是否关闭抢占式调度
	preemptoff    string // if != "", keep curg running on this m
	locks         int32
	dying         int32
	profilehz     int32
    //M的自旋状态，为true时M处于自旋状态，正在从其他线程偷G; 为false，休眠状态
	spinning      bool // m is out of work and is actively looking for work
	blocked       bool // m is blocked on a note
	newSigstack   bool // minit on C thread called sigaltstack
	printlock     int8
	incgo         bool   // m is executing a cgo call
	freeWait      uint32 // if == 0, safe to free g0 and delete m (atomic)
	fastrand      [2]uint32
	needextram    bool
	traceback     uint8
	ncgocall      uint64      // number of cgo calls in total
	ncgo          int32       // number of cgo calls currently in progress
	cgoCallersUse uint32      // if non-zero, cgoCallers in use temporarily
	cgoCallers    *cgoCallers // cgo traceback if crashing in cgo call
	//没有goroutine运行时，工作线程睡眠
    //通过这个来唤醒工作线程
    park          note // 休眠锁
    //记录所有工作线程的链表
	alllink       *m // on allm
	schedlink     muintptr
    //当前线程内存分配的本地缓存
	mcache        *mcache
    //当前M锁定的G，
	lockedg       guintptr
	createstack   [32]uintptr // stack that created this thread.
	lockedExt     uint32      // tracking for external LockOSThread
	lockedInt     uint32      // tracking for internal lockOSThread
	nextwaitm     muintptr    // next m waiting for lock
	waitunlockf   func(*g, unsafe.Pointer) bool
	waitlock      unsafe.Pointer
	waittraceev   byte
	waittraceskip int
	startingtrace bool
	syscalltick   uint32
    //操作系统线程id
	thread        uintptr // thread handle
	freelink      *m      // on sched.freem

	// these are here because they are too large to be on the stack
	// of low-level NOSPLIT functions.
	libcall   libcall
	libcallpc uintptr // for cpu profiler
	libcallsp uintptr
	libcallg  guintptr
	syscall   libcall // stores syscall parameters on windows

	vdsoSP uintptr // SP for traceback while in VDSO call (0 if not in call)
	vdsoPC uintptr // PC for traceback while in VDSO call

	dlogPerM

	mOS
}
```
看看几个比较重要的字段：

- g0：用于执行调度器的g0

- gsignal：用于信号处理

- tls：线程本地存储的tls

- p：goroutine绑定的本地资源



## P 结构体
一个 M 要运行，必须绑定 P 才能运行 goroutine，其中一个 M 阻塞时，P 会被传给其他 M 执行。

**P**（Processor），逻辑处理器，调度的核心中间层，持有运行 G 的资源（本地运行队列 runq、M 的绑定关系），只有绑定 P 的 M 才能执行 G，P 的数量由`GOMAXPROCS`控制（默认等于 CPU 核心数）。

/src/runtime/runtime2.go

```go
type p struct {
    //allp中的索引
	id          int32
    //p的状态
	status      uint32 // one of pidle/prunning/...
	link        puintptr
	schedtick   uint32     // incremented on every scheduler call->每次scheduler调用+1
	syscalltick uint32     // incremented on every system call->每次系统调用+1
	sysmontick  sysmontick // last tick observed by sysmon
    //指向绑定的 m，如果 p 是 idle 的话，那这个指针是 nil
	m           muintptr   // back-link to associated m (nil if idle)
	mcache      *mcache
	raceprocctx uintptr

    //不同大小可用defer结构池
	deferpool    [5][]*_defer // pool of available defer structs of different sizes (see panic.go)
	deferpoolbuf [5][32]*_defer

	// Cache of goroutine ids, amortizes accesses to runtime·sched.goidgen.
	goidcache    uint64
	goidcacheend uint64

    //本地运行队列，可以无锁访问
	// Queue of runnable goroutines. Accessed without lock.
	runqhead uint32  //队列头
	runqtail uint32   //队列尾
    //数组实现的循环队列
	runq     [256]guintptr
    
	// runnext, if non-nil, is a runnable G that was ready'd by
	// the current G and should be run next instead of what's in
	// runq if there's time remaining in the running G's time
	// slice. It will inherit the time left in the current time
	// slice. If a set of goroutines is locked in a
	// communicate-and-wait pattern, this schedules that set as a
	// unit and eliminates the (potentially large) scheduling
	// latency that otherwise arises from adding the ready'd
	// goroutines to the end of the run queue.
    // runnext 非空时，代表的是一个 runnable 状态的 G，
    //这个 G 被 当前 G 修改为 ready 状态，相比 runq 中的 G 有更高的优先级。
    //如果当前 G 还有剩余的可用时间，那么就应该运行这个 G
    //运行之后，该 G 会继承当前 G 的剩余时间
	runnext guintptr

	// Available G's (status == Gdead)
    //空闲的g
	gFree struct {
		gList
		n int32
	}

	sudogcache []*sudog
	sudogbuf   [128]*sudog

	tracebuf traceBufPtr

	// traceSweep indicates the sweep events should be traced.
	// This is used to defer the sweep start event until a span
	// has actually been swept.
	traceSweep bool
	// traceSwept and traceReclaimed track the number of bytes
	// swept and reclaimed by sweeping in the current sweep loop.
	traceSwept, traceReclaimed uintptr

	palloc persistentAlloc // per-P to avoid mutex

	_ uint32 // Alignment for atomic fields below

	// Per-P GC state
	gcAssistTime         int64    // Nanoseconds in assistAlloc
	gcFractionalMarkTime int64    // Nanoseconds in fractional mark worker (atomic)
	gcBgMarkWorker       guintptr // (atomic)
	gcMarkWorkerMode     gcMarkWorkerMode

	// gcMarkWorkerStartTime is the nanotime() at which this mark
	// worker started.
	gcMarkWorkerStartTime int64

	// gcw is this P's GC work buffer cache. The work buffer is
	// filled by write barriers, drained by mutator assists, and
	// disposed on certain GC state transitions.
	gcw gcWork

	// wbBuf is this P's GC write barrier buffer.
	//
	// TODO: Consider caching this in the running G.
	wbBuf wbBuf

	runSafePointFn uint32 // if 1, run sched.safePointFn at next safe point

	pad cpu.CacheLinePad
}
```
其他的一些字段就是gc，trace，debug信息



## G 结构体

G 就是 goroutine，轻量级协程，用户态执行单元。主要保存 goroutine 的所有信息以及栈信息，gobuf 结构体：cpu 里的寄存器信息，以便在轮到本 goroutine 执行时，知道从哪里开始执行。

G 是一个**可被调度的执行单元**。

包含：

- 执行栈
- 寄存器上下文
- 当前状态
- 调度信息

/src/runtime/runtime2.go

```go
type stack struct {
	lo uintptr   //栈顶，指向内存低地址
	hi uintptr   //栈底，指向内存搞地址
}

type g struct {
	// Stack parameters.
	// stack describes the actual stack memory: [stack.lo, stack.hi).
	// stackguard0 is the stack pointer compared in the Go stack growth prologue.
	// It is stack.lo+StackGuard normally, but can be StackPreempt to trigger a preemption.
	// stackguard1 is the stack pointer compared in the C stack growth prologue.
	// It is stack.lo+StackGuard on g0 and gsignal stacks.
	// It is ~0 on other goroutine stacks, to trigger a call to morestackc (and crash).
	// 记录该goroutine使用的栈
    stack       stack   // offset known to runtime/cgo
    
	//下面两个成员用于栈溢出检查，实现栈的自动伸缩，抢占调度也会用到stackguard0
    stackguard0 uintptr // offset known to liblink
	stackguard1 uintptr // offset known to liblink

	_panic         *_panic // innermost panic - offset known to liblink
	_defer         *_defer // innermost defer
    
    // 此goroutine正在被哪个工作线程执行
	m              *m      // current m; offset known to arm liblink
    //这个字段跟调度切换有关，G切换时用来保存上下文，保存什么，看下面gobuf结构体
	sched          gobuf
	syscallsp      uintptr        // if status==Gsyscall, syscallsp = sched.sp to use during gc
	syscallpc      uintptr        // if status==Gsyscall, syscallpc = sched.pc to use during gc
	stktopsp       uintptr        // expected sp at top of stack, to check in traceback
	param          unsafe.Pointer // passed parameter on wakeup，wakeup唤醒时传递的参数
	// 状态Gidle,Grunnable,Grunning,Gsyscall,Gwaiting,Gdead
    atomicstatus   uint32
	stackLock      uint32 // sigprof/scang lock; TODO: fold in to atomicstatus
	goid           int64
    
    //schedlink字段指向全局运行队列中的下一个g，
    //所有位于全局运行队列中的g形成一个链表
	schedlink      guintptr
	waitsince      int64      // approx time when the g become blocked
	waitreason     waitReason // if status==Gwaiting，g被阻塞的原因
    //抢占信号，stackguard0 = stackpreempt，如果需要抢占调度，设置preempt为true
	preempt        bool       // preemption signal, duplicates stackguard0 = stackpreempt
	paniconfault   bool       // panic (instead of crash) on unexpected fault address
	preemptscan    bool       // preempted g does scan for gc
	gcscandone     bool       // g has scanned stack; protected by _Gscan bit in status
	gcscanvalid    bool       // false at start of gc cycle, true if G has not run since last scan; TODO: remove?
	throwsplit     bool       // must not split stack
	raceignore     int8       // ignore race detection events
	sysblocktraced bool       // StartTrace has emitted EvGoInSyscall about this goroutine
	sysexitticks   int64      // cputicks when syscall has returned (for tracing)
	traceseq       uint64     // trace event sequencer
	tracelastp     puintptr   // last P emitted an event for this goroutine
	// 如果调用了 LockOsThread，那么这个 g 会绑定到某个 m 上
    lockedm        muintptr
	sig            uint32
	writebuf       []byte
	sigcode0       uintptr
	sigcode1       uintptr
	sigpc          uintptr
    // 创建这个goroutine的go表达式的pc
	gopc           uintptr         // pc of go statement that created this goroutine
	ancestors      *[]ancestorInfo // ancestor information goroutine(s) that created this goroutine (only used if debug.tracebackancestors)
	startpc        uintptr         // pc of goroutine function
	racectx        uintptr
	waiting        *sudog         // sudog structures this g is waiting on (that have a valid elem ptr); in lock order
	cgoCtxt        []uintptr      // cgo traceback context
	labels         unsafe.Pointer // profiler labels
	timer          *timer         // cached timer for time.Sleep,为 time.Sleep 缓存的计时器
	selectDone     uint32         // are we participating in a select and did someone win the race?

	// Per-G GC state

	// gcAssistBytes is this G's GC assist credit in terms of
	// bytes allocated. If this is positive, then the G has credit
	// to allocate gcAssistBytes bytes without assisting. If this
	// is negative, then the G must correct this by performing
	// scan work. We track this in bytes to make it fast to update
	// and check for debt in the malloc hot path. The assist ratio
	// determines how this corresponds to scan work debt.
	gcAssistBytes int64
}
```

### gobuf 调度栈信息
gobuf 结构体用于保存 goroutine 的调度栈信息，主要包括CPU的几个寄存器的值。

要了解寄存器是什么，可以点击这里：
[寄存器1](https://zh.wikipedia.org/wiki/%E5%AF%84%E5%AD%98%E5%99%A8)
[寄存器2](https://baike.baidu.com/item/%E5%AF%84%E5%AD%98%E5%99%A8)

/src/runtime/runtime2.go
```go
type gobuf struct {
	// The offsets of sp, pc, and g are known to (hard-coded in) libmach.
	//
	// ctxt is unusual with respect to GC: it may be a
	// heap-allocated funcval, so GC needs to track it, but it
	// needs to be set and cleared from assembly, where it's
	// difficult to have write barriers. However, ctxt is really a
	// saved, live register, and we only ever exchange it between
	// the real register and the gobuf. Hence, we treat it as a
	// root during stack scanning, which means assembly that saves
	// and restores it doesn't need write barriers. It's still
	// typed as a pointer so that any other writes from Go get
	// write barriers.
	sp   uintptr      // 保存CPU的rsp寄存器的值
	pc   uintptr      // 保存CPU的rip寄存器的值
	g    guintptr     // 记录当前这个gobuf对象属于哪个goroutine
	ctxt unsafe.Pointer
    
    //保存系统调用的返回值，因为从系统调用返回之后如果p被其它工作线程抢占，
    //则这个goroutine会被放入全局运行队列被其它工作线程调度，其它线程需要知道系统调用的返回值。
	ret  sys.Uintreg  // 保存系统调用的返回值
	lr   uintptr
    
    //保存CPU的rip寄存器的值
	bp   uintptr // for GOEXPERIMENT=framepointer
}
```


## 调度器sched结构

所有的gorouteine都是被调度器调度运行，调度器持有全局资源

### schedt 全局调度器

schedt，全局调度器，调度器的总控中心，管理全局资源。

/src/runtime/runtime2.go
```go
type schedt struct {
	// accessed atomically. keep at top to ensure alignment on 32-bit systems.
    // 需以原子访问访问。
    // 保持在 struct 顶部，以使其在 32 位系统上可以对齐
	goidgen  uint64
	lastpoll uint64

	lock mutex

	// When increasing nmidle, nmidlelocked, nmsys, or nmfreed, be
	// sure to call checkdead().
	
    //由空闲的工作线程组成的链表
	midle        muintptr // idle m's waiting for work
    //空闲的工作线程的数量
	nmidle       int32    // number of idle m's waiting for work
    //空闲的且被 lock 的 m 计数
	nmidlelocked int32    // number of locked m's waiting for work
    //已经创建的多个m，下一个m id
	mnext        int64    // number of m's that have been created and next M ID
    //被允许创建的最大m线程数量
	maxmcount    int32    // maximum number of m's allowed (or die)
	nmsys        int32    // number of system m's not counted for deadlock
    //累积空闲的m数量
	nmfreed      int64    // cumulative number of freed m's

    //系统goroutine的数量，自动更新
	ngsys uint32 // number of system goroutines; updated atomically
	
    //由空闲的 p 结构体对象组成的链表
	pidle      puintptr // idle p's
    //空闲的 p 结构体对象的数量
	npidle     uint32
	nmspinning uint32 // See "Worker thread parking/unparking" comment in proc.go.

	// Global runnable queue.
    //全局运行队列 G队列
	runq     gQueue //这个结构体在proc.go里
    //元素数量
	runqsize int32

	// disable controls selective disabling of the scheduler.
	//
	// Use schedEnableUser to control this.
	//
	// disable is protected by sched.lock.
	disable struct {
		// user disables scheduling of user goroutines.
		user     bool
		runnable gQueue // pending runnable Gs
		n        int32  // length of runnable
	}

	// Global cache of dead G's. 有效 dead G 全局缓存
	gFree struct {
		lock    mutex
		stack   gList // Gs with stacks
		noStack gList // Gs without stacks
		n       int32
	}

	// Central cache of sudog structs. sudog结构的集中缓存
	sudoglock  mutex
	sudogcache *sudog

	// Central pool of available defer structs of different sizes. 不同大小有效的defer结构的池
	deferlock mutex
	deferpool [5]*_defer

	// freem is the list of m's waiting to be freed when their
	// m.exited is set. Linked through m.freelink.
	freem *m

	gcwaiting  uint32 // gc is waiting to run
	stopwait   int32
	stopnote   note
	sysmonwait uint32
	sysmonnote note

	// safepointFn should be called on each P at the next GC
	// safepoint if p.runSafePointFn is set.
	safePointFn   func(*p)
	safePointWait int32
	safePointNote note

	profilehz int32 // cpu profiling rate

	procresizetime int64 // nanotime() of last change to gomaxprocs
	totaltime      int64 // ∫gomaxprocs dt up to procresizetime
}
```


### gQueue 全局 G 队列
/src/runtime/proc.go

```go
type gQueue struct {
	head guintptr //队列头
	tail guintptr //队列尾
}
```



## 一些重要全局变量
/src/runtime/proc.go
```go
m0 m            //代表主线程
g0  g          //m0绑定的g0，也就是M结构体中m0.g0=&g0


allgs  []*g  //保存所有的g
```

关键特殊 G：`g0`是 M 的调度协程，无用户代码，使用系统栈（不参与 GC），所有调度操作（G 切换、栈扩缩容）都在`g0`的上下文中执行。

/src/runtime/runtime2.go

```go
allm  *m             //所有的m构成的一个链表，包括上面的m0
allp  []*p            //保存所有的p， len(allp) == gomaxprocs

sched         schedt //调度器的结构体，保存了调度器的各种信息

ncpu       int32  //系统cpu核的数量，程序启动时由runtime初始化
gomaxprocs int32 //p 的最大数量，默认等于ncpu，可以通过GOMAXPROCS修改
```


在程序初始化时，这些变量都会被初始化为0值，指针会被初始化为nil指针，切片初始化为nil切片，int被初始化为数字0，结构体的所有成员变量按其本类型初始化为其类型的0值。

## GPM + OS 层级运行关系

从上层 Go 运行时到下层 OS 内核，核心关系图：

```bash
┌──────────────────────────────── Go 应用层 ───────────────────────────────┐
│  无数用户Goroutine（G1、G2、G3...）| 开发者编写的业务代码/库代码               │
└───────────────────────────┬─────────────────────────────────────────────┘
                            │ 被 P 调度，从 P 的本地/全局队列获取
                            |
┌───────────────────────────▼─────────────────────────────────────────────┐
│  P（逻辑处理器，数量=GOMAXPROCS）| 持有本地G队列+mcache，是调度核心            │
│  [P0] [P1] [P2] ... [Pn-1]                                              │
│   ▲    ▲    ▲          ▲                                                │
│   │    │    │          │  绑定/解绑（M获取P才能执行G，P可被抢占/移交）
┌───▼────▼────▼──────────▼───────────────────────────────────────────────┐
│  M（内核线程封装，数量动态变化）| 真正执行指令，绑定P后成为“工作M”               │
│  [M0] [M1] [M2] ... [Mx]（x≥n，存在空闲M池）                              │
│   ▲    ▲    ▲          ▲                                               │
│   │    │    │          │  一对一映射（M封装OS线程，由OS调度到CPU）
┌───▼────▼────▼──────────▼───────────────────────────────────────────────┐
│  OS 内核层                                                              │
│  [OS Thread0] [OS Thread1] [OS Thread2] ... [OS Threadx]               │
│   ▲    ▲        ▲          ▲                                           │
│   │    │        │          │  OS内核的CPU调度（时间片/多核并行）
└───▼────▼────────▼──────────▼────────────────────────────────────────────┘
│  物理CPU核心（Core0、Core1、Core2...）| 最终的指令执行硬件                     │
└──────────────────────────────────────────────────────────────────────────┘
```

核心运行关系规则：

1. **P 是核心职责**：M 必须绑定 P（`m.p = p`）才能执行 G，无 P 的 M 只能进入空闲链表。
2. **P 的独占性**：一个 P 同一时间只能绑定一个 M，一个 M 同一时间只能绑定一个 P；但 P 可在 M 之间移交（如 M 进入系统调用，P 会被剥离给其他空闲 M）。
3. **OS 的底层支撑**：OS 负责将内核线程调度到物理 CPU 核心，Go 运行时不干预 OS 的 CPU 调度，仅在用户态做 G 的调度。
4. **多核心并行**：P 的数量 =`GOMAXPROCS`，决定了 Go 程序最大并行执行 G 的数量（多核下，每个 P 对应一个核心的执行流）。

## 调度器初始化
调度器初始化有一个主要的函数 `schedinit()`， 这个函数在 `/src/runtime/proc.go` 文件中。

函数开头还把初始化的顺序给列出来了：

> _// The bootstrap sequence is:_
> _//_
> _//  call osinit_
> _//  call schedinit_
> _//  make & queue new G_
> _//  call runtime·mstart_
> _//_
> _// The new G calls runtime·main._


```go
func schedinit() {
	// raceinit must be the first call to race detector.
	// In particular, it must be done before mallocinit below calls racemapshadow.
	_g_ := getg() //getg() 在 src/runtime/stubs.go 中声明，真正的代码由编译器生成
	if raceenabled {
		_g_.racectx, raceprocctx0 = raceinit()
	}
	
    //设置最大M的数量
	sched.maxmcount = 10000

	tracebackinit()
	moduledataverify()
    //初始化栈空间常用管理链表
	stackinit()
	mallocinit()
    //初始化当前m
	mcommoninit(_g_.m)
	cpuinit()       // must run before alginit
	alginit()       // maps must not be used before this call
	modulesinit()   // provides activeModules
	typelinksinit() // uses maps, activeModules
	itabsinit()     // uses activeModules

	msigsave(_g_.m)
	initSigmask = _g_.m.sigmask

	goargs()
	goenvs()
	parsedebugvars()
	gcinit()

	sched.lastpoll = uint64(nanotime())
    // 把p数量从1调整到默认的CPU Core数量
	procs := ncpu
	if n, ok := atoi32(gogetenv("GOMAXPROCS")); ok && n > 0 {
		procs = n
	}
    //调整P数量
    //这里的P都是新建的，所以不返回有本地任务的p
	if procresize(procs) != nil {
		throw("unknown runnable goroutine during bootstrap")
	}

	// For cgocheck > 1, we turn on the write barrier at all times
	// and check all pointer writes. We can't do this until after
	// procresize because the write barrier needs a P.
	if debug.cgocheck > 1 {
		writeBarrier.cgo = true
		writeBarrier.enabled = true
		for _, p := range allp {
			p.wbBuf.reset()
		}
	}

	if buildVersion == "" {
		// Condition should never trigger. This code just serves
		// to ensure runtime·buildVersion is kept in the resulting binary.
		buildVersion = "unknown"
	}
	if len(modinfo) == 1 {
		// Condition should never trigger. This code just serves
		// to ensure runtime·modinfo is kept in the resulting binary.
		modinfo = ""
	}
}
```

开头的这个函数getg()，跳转到了 func getg() *g  ，定义这么一个形式，什么意思？

函数首先调用 `getg()` 函数获取当前正在运行的 `g`，`getg()` 在 `src/runtime/stubs.go` 中声明，真正的代码由编译器生成。

```go
// getg returns the pointer to the current g.
// The compiler rewrites calls to this function into instructions
// that fetch the g directly (from TLS or from the dedicated register).
func getg() *g
```
注释里也说了，getg 返回当前正在运行的 goroutine 的指针，它会从 tls 里取出 tls[0]，也就是当前运行的 goroutine 的地址。编译器插入类似下面的代码:
```go
get_tls(CX) 
MOVQ g(CX), BX; // BX存器里面现在放的是当前g结构体对象的地址
```
原来是这么个意思。

**调度器初始化大致过程：**


M初始化            -->   P 初始化          - -> G初始化

mcommoninit           Procresize                newproc

-------------------------------------------------------

allm 池                     allp池                       g.sched执行现场

​                                                               p.runq 调度队列

MPG初始化过程。 M/P/G 初始化：mcommoninit、procresize、newproc，他们负责M资源池（allm）、p资源池（allp）、G的运行现场（g.sched） 以及调度队列（p.runq）

  

## 调度循环
所有的工作初始化完成后，就要启动运行器了。准备工作做好了，就要启动mstart了。

这个工作在汇编语言中也可以看出来

/src/runtime/asm_amd64.s  (在linux下)
```bash
TEXT runtime·rt0_go(SB),NOSPLIT,$0

  ... ... ...

  MOVL	16(SP), AX		// copy argc
	MOVL	AX, 0(SP)
	MOVQ	24(SP), AX		// copy argv
	MOVQ	AX, 8(SP)
	CALL	runtime·args(SB)  
	CALL	runtime·osinit(SB)    //OS初始化
	CALL	runtime·schedinit(SB) //调度器初始化

	// create a new goroutine to start program
	MOVQ	$runtime·mainPC(SB), AX		// entry
	PUSHQ	AX
	PUSHQ	$0			// arg size
	CALL	runtime·newproc(SB)       // G 初始化
	POPQ	AX
	POPQ	AX

	// start this M , 启动M
	CALL	runtime·mstart(SB)

	CALL	runtime·abort(SB)	// mstart should never return
	RET
```



## 参考

1. 雨痕 《Go语言学习笔记》 [https://book.douban.com/subject/26832468/](https://book.douban.com/subject/26832468/) 
2. 深度解密Go语言 [https://qcrao.com/2019/09/02/dive-into-go-scheduler/](https://qcrao.com/2019/09/02/dive-into-go-scheduler/)
3. https://blog.csdn.net/u010853261/article/details/84790392