// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package g

import (
	"unsafe"

	"github.com/sitano/gsysint/runtime/sys"
)

// defined constants
const (
	// G status
	//
	// Beyond indicating the general state of a G, the G status
	// acts like a lock on the goroutine's stack (and hence its
	// ability to execute user code).
	//
	// If you add to this list, add to the list
	// of "okay during garbage collection" status
	// in mgcmark.go too.

	// _Gidle means this goroutine was just allocated and has not
	// yet been initialized.
	_Gidle = iota // 0

	// _Grunnable means this goroutine is on a run queue. It is
	// not currently executing user code. The stack is not owned.
	_Grunnable // 1

	// _Grunning means this goroutine may execute user code. The
	// stack is owned by this goroutine. It is not on a run queue.
	// It is assigned an M and a P.
	_Grunning // 2

	// _Gsyscall means this goroutine is executing a system call.
	// It is not executing user code. The stack is owned by this
	// goroutine. It is not on a run queue. It is assigned an M.
	_Gsyscall // 3

	// _Gwaiting means this goroutine is blocked in the runtime.
	// It is not executing user code. It is not on a run queue,
	// but should be recorded somewhere (e.g., a channel wait
	// queue) so it can be ready()d when necessary. The stack is
	// not owned *except* that a channel operation may read or
	// write parts of the stack under the appropriate channel
	// lock. Otherwise, it is not safe to access the stack after a
	// goroutine enters _Gwaiting (e.g., it may get moved).
	_Gwaiting // 4

	// _Gmoribund_unused is currently unused, but hardcoded in gdb
	// scripts.
	_Gmoribund_unused // 5

	// _Gdead means this goroutine is currently unused. It may be
	// just exited, on a free list, or just being initialized. It
	// is not executing user code. It may or may not have a stack
	// allocated. The G and its stack (if any) are owned by the M
	// that is exiting the G or that obtained the G from the free
	// list.
	_Gdead // 6

	// _Genqueue_unused is currently unused.
	_Genqueue_unused // 7

	// _Gcopystack means this goroutine's stack is being moved. It
	// is not executing user code and is not on a run queue. The
	// stack is owned by the goroutine that put it in _Gcopystack.
	_Gcopystack // 8

	// _Gpreempted means this goroutine stopped itself for a
	// suspendG preemption. It is like _Gwaiting, but nothing is
	// yet responsible for ready()ing it.
	_Gpreempted // 9

	// _Gleaked represents a leaked goroutine caught by the GC.
	_Gleaked // 10

	// _Gscan combined with one of the above states other than
	// _Grunning indicates that GC is scanning the stack. The
	// goroutine is not executing user code and the stack is owned
	// by the goroutine that set the _Gscan bit.
	//
	// _Gscanrunning is different: it is used to briefly block
	// state transitions while GC signals the G to scan its own
	// stack. This is otherwise like _Grunning.
	//
	// atomicstatus&~Gscan gives the state the goroutine will
	// return to when the scan completes.
	_Gscan          = 0x1000
	_Gscanrunnable  = _Gscan + _Grunnable  // 0x1001
	_Gscanrunning   = _Gscan + _Grunning   // 0x1002
	_Gscansyscall   = _Gscan + _Gsyscall   // 0x1003
	_Gscanwaiting   = _Gscan + _Gwaiting   // 0x1004
	_Gscanpreempted = _Gscan + _Gpreempted // 0x1009
)

const (
	// P status
	_Pidle    = iota
	_Prunning // Only this P is allowed to change from _Prunning.
	_Psyscall_unused
	_Pgcstop
	_Pdead
)

const (
	MutexUnlocked = 0
	MutexLocked   = 1
	MutexSleeping = 2
)

// Mutual exclusion locks.  In the uncontended case,
// as fast as spin locks (just a few user-level instructions),
// but on the contention path they sleep in the kernel.
// A zeroed Mutex is unlocked (no need to initialize each lock).
type Mutex struct {
	// Empty struct if lock ranking is disabled, otherwise includes the lock rank
	lockRankStruct
	// Futex-based impl treats it as uint32 key,
	// while sema-based impl as M* waitm.
	// Used to be a union, but unions break precise GC.
	Key uintptr
}

// sleep and wakeup on one-time events.
// before any calls to notesleep or notewakeup,
// must call noteclear to initialize the Note.
// then, exactly one thread can call notesleep
// and exactly one thread can call notewakeup (once).
// once notewakeup has been called, the notesleep
// will return.  future notesleep will return immediately.
// subsequent noteclear must be called only after
// previous notesleep has returned, e.g. it's disallowed
// to call noteclear straight after notewakeup.
//
// notetsleep is like notesleep but wakes up after
// a given number of nanoseconds even if the event
// has not yet happened.  if a goroutine uses notetsleep to
// wake up early, it must wait to call noteclear until it
// can be sure that no other goroutine is calling
// notewakeup.
//
// notesleep/notetsleep are generally called on g0,
// notetsleepg is similar to notetsleep but is called on user g.
type Note struct {
	// Futex-based impl treats it as uint32 key,
	// while sema-based impl as M* waitm.
	// Used to be a union, but unions break precise GC.
	Key uintptr
}

type FuncVal struct {
	FN uintptr
	// variable-size, fn-specific data here
}

// The guintptr, muintptr, and puintptr are all used to bypass write barriers.
// It is particularly important to avoid write barriers when the current P has
// been released, because the GC thinks the world is stopped, and an
// unexpected write barrier would not be synchronized with the GC,
// which can lead to a half-executed write barrier that has marked the object
// but not queued it. If the GC skips the object and completes before the
// queuing can occur, it will incorrectly free the object.
//
// We tried using special assignment functions invoked only when not
// holding a running P, but then some updates to a particular memory
// word went through write barriers and some did not. This breaks the
// write barrier shadow checking mode, and it is also scary: better to have
// a word that is completely ignored by the GC than to have one for which
// only a few updates are ignored.
//
// Gs and Ps are always reachable via true pointers in the
// allgs and allp lists or (during allocation before they reach those lists)
// from stack variables.
//
// Ms are always reachable via true pointers either from allm or
// freem. Unlike Gs and Ps we do free Ms, so it's important that
// nothing ever hold an muintptr across a safe point.

// A guintptr holds a goroutine pointer, but typed as a uintptr
// to bypass write barriers. It is used in the Gobuf goroutine state
// and in scheduling lists that are manipulated without a P.
//
// The Gobuf.g goroutine pointer is almost always updated by assembly code.
// In one of the few places it is updated by Go code - func save - it must be
// treated as a uintptr to avoid a write barrier being emitted at a bad time.
// Instead of figuring out how to emit the write barriers missing in the
// assembly manipulation, we change the type of the field to uintptr,
// so that it does not require write barriers at all.
//
// Goroutine structs are published in the allg list and never freed.
// That will keep the goroutine structs from being collected.
// There is never a time that Gobuf.g's contain the only references
// to a goroutine: the publishing of the goroutine in allg comes first.
// Goroutine pointers are also kept in non-GC-visible places like TLS,
// so I can't see them ever moving. If we did want to start moving data
// in the GC, we'd need to allocate the goroutine structs from an
// alternate arena. Using guintptr doesn't make that problem any worse.
type Guintptr uintptr

//go:nosplit
func (gp Guintptr) Ptr() *G { return (*G)(unsafe.Pointer(gp)) }

//go:nosplit
func (gp *Guintptr) Set(g *G) { *gp = Guintptr(unsafe.Pointer(g)) }

//func (gp *guintptr) Cas(old, new guintptr) bool {
//	return atomic.Casuintptr((*uintptr)(unsafe.Pointer(gp)), uintptr(old), uintptr(new))
//}

type Puintptr uintptr

//func (pp Puintptr) ptr() *P { return (*P)(unsafe.Pointer(pp)) }

//func (pp *Puintptr) set(p *P) { *pp = Puintptr(unsafe.Pointer(p)) }

// muintptr is a *m that is not tracked by the garbage collector.
//
// Because we do free Ms, there are some additional constrains on
// muintptrs:
//
// 1. Never hold an muintptr locally across a safe point.
//
//  2. Any muintptr in the heap must be owned by the M itself so it can
//     ensure it is not in use when the last true *m is released.
type Muintptr uintptr

//go:nosplit
func (mp Muintptr) Ptr() *M { return (*M)(unsafe.Pointer(mp)) }

//go:nosplit
func (mp *Muintptr) Set(m *M) { *mp = Muintptr(unsafe.Pointer(m)) }

type Uintreg = sys.Uintreg

type GoBuf struct {
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
	sp   uintptr
	pc   uintptr
	g    Guintptr
	ctxt unsafe.Pointer
	lr   uintptr
	bp   uintptr // for framepointer-enabled architectures
}

// maybeTraceablePtr is a pointer that is traceable by the GC.
// (Under the race detector or in other special builds it is
// the source of truth for the pointer value.)
type maybeTraceablePtr struct {
	vp unsafe.Pointer // For liveness only.
	vu uintptr        // Source of truth.
}

// Sudog represents a g in a wait list, such as for sending/receiving
// on a channel.
//
// Sudog is necessary because the g ↔ synchronization object relation
// is many-to-many. a g can be on many wait lists, so there may be
// many sudogs for one g; and many gs may be waiting on the same
// synchronization object, so there may be many sudogs for one object.
//
// Sudogs are allocated from a special pool. use acquiresudog and
// releasesudog to allocate and free them.
type Sudog struct {
	// The following fields are protected by the hchan.lock of the
	// channel this sudog is blocking on. shrinkstack depends on
	// this for sudogs involved in channel ops.

	g *G

	next *Sudog
	prev *Sudog

	elem maybeTraceablePtr // data element (may point to stack)

	// The following fields are never accessed concurrently.
	// For channels, waitlink is only accessed by g.
	// For semaphores, all fields (including the ones above)
	// are only accessed when holding a semaRoot lock.

	acquiretime int64
	releasetime int64
	ticket      uint32

	// isSelect indicates g is participating in a select, so
	// g.selectDone must be CAS'd to win the wake-up race.
	isSelect bool

	// success indicates whether communication over channel c
	// succeeded. It is true if the goroutine was awoken because a
	// value was delivered over channel c, and false if awoken
	// because c was closed.
	success bool

	// waiters is a count of semaRoot waiting list other than head of list,
	// clamped to a uint16 to fit in unused space.
	waiters uint16

	parent   *Sudog            // semaRoot binary tree
	waitlink *Sudog            // g.waiting list or semaRoot
	waittail *Sudog            // semaRoot
	c        maybeTraceablePtr // channel
}

// Stack describes a Go execution stack.
// The bounds of the stack are exactly [lo, hi),
// with no implicit data structures on either side.
type Stack struct {
	lo uintptr
	hi uintptr
}

// Defer is a deferred call record.
type Defer struct {
	heap      bool
	rangefunc bool    // true for rangefunc list
	sp        uintptr // sp at time of defer
	pc        uintptr // pc at time of defer
	fn        func()  // can be nil for open-coded defers
	link      *Defer  // next defer on G; can point to either heap or stack!

	head unsafe.Pointer // *atomic.Pointer[_defer]
}

// Panic is a panic record.
type Panic struct {
	arg  any    // argument to panic
	link *Panic // link to earlier panic

	startPC uintptr
	startSP unsafe.Pointer

	pc uintptr
	sp unsafe.Pointer
	fp unsafe.Pointer

	retpc uintptr

	deferBitsPtr *uint8
	slotsPtr     unsafe.Pointer

	recovered   bool // whether this panic has been recovered
	repanicked  bool // whether this panic repanicked
	goexit      bool
	deferreturn bool
}

// AncestorInfo records details of where a goroutine was started.
type AncestorInfo struct {
	PCS  []uintptr // pcs from the stack of this goroutine
	GoID uint64    // goroutine id of this goroutine; original goroutine possibly dead
	GoPC uintptr   // pc of go statement that created this goroutine
}

// G is the goroutine descriptor of go1.27.
//
// Only a prefix of the runtime struct is mirrored: the tail (xRegs,
// trace, gcAssistBytes, valgrindStackID) is left out because xRegs depends
// on the GOARCH/GOEXPERIMENT. Never allocate or copy a G, only use
// pointers received from the runtime.
type G struct {
	Stack       Stack   // offset known to runtime/cgo
	StackGuard0 uintptr // offset known to cmd/internal/obj/*
	StackGuard1 uintptr // offset known to cmd/internal/obj/*

	Panic     *Panic // innermost panic
	Defer     *Defer // innermost defer
	M         *M     // current m; offset known to arm liblink
	Sched     GoBuf
	SysCallSP uintptr // if status==Gsyscall, syscallsp = sched.sp to use during gc
	SysCallPC uintptr // if status==Gsyscall, syscallpc = sched.pc to use during gc
	SysCallBP uintptr // if status==Gsyscall, syscallbp = sched.bp to use in fpTraceback
	StkTopSP  uintptr // expected sp at top of stack, to check in traceback

	Param        unsafe.Pointer // passed parameter on wakeup
	AtomicStatus uint32
	StackLock    uint32 // sigprof/scang lock; TODO: fold in to atomicstatus
	GoID         uint64
	SchedLink    Guintptr
	WaitSince    int64      // approx time when the g become blocked
	WaitReason   WaitReason // if status==Gwaiting

	Preempt       bool // preemption signal, duplicates stackguard0 = stackpreempt
	PreemptStop   bool // transition to _Gpreempted on preemption; otherwise, just deschedule
	PreemptShrink bool // shrink stack at synchronous safe point

	// AsyncSafePoint is set if g is stopped at an asynchronous
	// safe point. This means there are frames on the stack
	// without precise pointer information.
	AsyncSafePoint bool

	PanicOnFault     bool // panic (instead of crash) on unexpected fault address
	GcScanDone       bool // g has scanned stack; protected by _Gscan bit in status
	ThrowSplit       bool // must not split stack
	ActiveStackChans bool
	ParkingOnChan    uint8
	InMarkAssist     bool
	CoroExit         bool // argument to coroswitch_m

	RaceIgnore      int8  // ignore race detection events
	NoCgoCallback   bool  // whether disable callback from C
	Tracking        bool  // whether we're tracking this G for sched latency statistics
	TrackingSeq     uint8 // used to decide whether to track this G
	TrackingStamp   int64 // timestamp of when the G last started being tracked
	RunnableTime    int64 // the amount of time spent runnable, cleared when running, only used when tracking
	LockedM         Muintptr
	FipsIndicator   uint8
	FipsOnlyBypass  bool
	DitWanted       bool // set if g wants to be executed with DIT enabled
	SyncSafePoint   bool // set if g is stopped at a synchronous safe point.
	RunningCleanups uint8
	Sig             uint32
	Secret          int32 // current nesting of runtime/secret.Do calls.
	WriteBuf        []byte
	SigCode0        uintptr
	SigCode1        uintptr
	SigPC           uintptr
	ParentGoID      uint64          // goid of goroutine that created this goroutine
	GoPC            uintptr         // pc of go statement that created this goroutine
	Ancestors       *[]AncestorInfo // ancestor information goroutine(s) that created this goroutine (only used if debug.tracebackancestors)
	StartPC         uintptr         // pc of goroutine function
	RaceCtx         uintptr
	Waiting         *Sudog         // sudog structures this g is waiting on (that have a valid elem ptr); in lock order
	CgoCtxt         []uintptr      // cgo traceback context
	Labels          unsafe.Pointer // profiler labels
	Timer           *Timer         // cached timer for time.Sleep
	SleepWhen       int64          // when to sleep until
	SelectDone      uint32         // are we participating in a select and did someone win the race?

	GoroutineProfiled uint32         // goroutineProfileStateHolder
	CoroArg           unsafe.Pointer // *coro, argument during coroutine transfers
	Bubble            unsafe.Pointer // *synctestBubble
}

// ListNodeManual is a node in a manually managed list.
type ListNodeManual struct {
	prev uintptr
	next uintptr
}

// M is the OS thread descriptor of go1.27.
//
// Only a prefix of the runtime struct is mirrored, up to LockedInt: the
// rest (mWaitList, mLockProfile, ..., mOS, ...) is large and OS specific.
// Never allocate or copy an M, only use pointers received from the runtime.
// SigMask is exact on darwin and linux only (see sigset_*.go).
type M struct {
	G0      *G     // goroutine with scheduling stack
	MoreBuf GoBuf  // gobuf arg to morestack
	DivMod  uint32 // div/mod denominator for arm - known to liblink (cmd/internal/obj/arm/obj5.go)

	// Fields whose offsets are not known to debuggers.

	ProcID     uint64       // for debuggers, but offset not hard-coded
	GSignal    *G           // signal-handling g
	GoSigStack GSignalStack // Go-allocated signal handling stack
	SigMask    SigSet       // storage for saved signal mask
	TLS        [6]uintptr   // thread-local storage (for x86 extern register)
	MStartFn   func()
	CurG       *G       // current running goroutine
	CaughtSig  Guintptr // goroutine running during fatal signal

	// Indicates whether we've received a signal while running in secret mode.
	SignalSecret bool

	// P is the currently attached P for executing Go code, nil if not executing user Go code.
	P               Puintptr
	NextP           Puintptr // The next P to install before executing. Implies exclusive ownership of this P.
	OldP            Puintptr // The P that was attached before executing a syscall.
	ID              int64
	MAllocing       int32
	Throwing        uint32 // throwType
	PreemptOff      string // if != "", keep curg running on this m
	Locks           int32
	Dying           int32
	ProfileHz       int32
	Spinning        bool // m is out of work and is actively looking for work
	Blocked         bool // m is blocked on a note
	NewSigStack     bool // minit on C thread called sigaltstack
	PrintLock       int8
	IncGo           bool   // m is executing a cgo call
	IsExtra         bool   // m is an extra m
	IsExtraInC      bool   // m is an extra m that does not have any Go frames
	IsExtraInSig    bool   // m is an extra m in a signal handler
	FreeWait        uint32 // Whether it is safe to free g0 and delete m (one of freeMRef, freeMStack, freeMWait)
	NeedExtraM      bool
	G0StackAccurate bool // whether the g0 stack has accurate bounds
	TraceBack       uint8
	AllpSnapshot    []unsafe.Pointer // []*p, snapshot of allp for use after dropping P in findRunnable, nil otherwise.
	NCgoCall        uint64           // number of cgo calls in total
	NCgo            int32            // number of cgo calls currently in progress
	CgoCallersUse   uint32           // if non-zero, cgoCallers in use temporarily
	CgoCallers      *CgoCallers      // cgo traceback if crashing in cgo call
	Park            Note
	AllLink         *M // on allm
	SchedLink       Muintptr
	IdleNode        ListNodeManual
	LockedG         Guintptr
	CreateStack     [32]uintptr // stack that created this thread, it's used for StackRecord.Stack0, so it must align with it.
	LockedExt       uint32      // tracking for external LockOSThread
	LockedInt       uint32      // tracking for internal lockOSThread
}

// A WaitReason explains why a goroutine has been stopped.
// See gopark. Do not re-use waitReasons, add new ones.
type WaitReason uint8

const (
	WaitReasonZero                  WaitReason = iota // ""
	WaitReasonGCAssistMarking                         // "GC assist marking"
	WaitReasonIOWait                                  // "IO wait"
	WaitReasonDumpingHeap                             // "dumping heap"
	WaitReasonGarbageCollection                       // "garbage collection"
	WaitReasonGarbageCollectionScan                   // "garbage collection scan"
	WaitReasonPanicWait                               // "panicwait"
	WaitReasonGCAssistWait                            // "GC assist wait"
	WaitReasonGCSweepWait                             // "GC sweep wait"
	WaitReasonGCScavengeWait                          // "GC scavenge wait"
	WaitReasonFinalizerWait                           // "finalizer wait"
	WaitReasonForceGCIdle                             // "force gc (idle)"
	WaitReasonUpdateGOMAXPROCSIdle                    // "GOMAXPROCS updater (idle)"
	WaitReasonSemacquire                              // "semacquire"
	WaitReasonSleep                                   // "sleep"
	WaitReasonChanReceiveNilChan                      // "chan receive (nil chan)"
	WaitReasonChanSendNilChan                         // "chan send (nil chan)"
	WaitReasonSelectNoCases                           // "select (no cases)"
	WaitReasonSelect                                  // "select"
	WaitReasonChanReceive                             // "chan receive"
	WaitReasonChanSend                                // "chan send"
	WaitReasonSyncCondWait                            // "sync.Cond.Wait"
	WaitReasonSyncMutexLock                           // "sync.Mutex.Lock"
	WaitReasonSyncRWMutexRLock                        // "sync.RWMutex.RLock"
	WaitReasonSyncRWMutexLock                         // "sync.RWMutex.Lock"
	WaitReasonSyncWaitGroupWait                       // "sync.WaitGroup.Wait"
	WaitReasonTraceReaderBlocked                      // "trace reader (blocked)"
	WaitReasonWaitForGCCycle                          // "wait for GC cycle"
	WaitReasonGCWorkerIdle                            // "GC worker (idle)"
	WaitReasonGCWorkerActive                          // "GC worker (active)"
	WaitReasonPreempted                               // "preempted"
	WaitReasonDebugCall                               // "debug call"
	WaitReasonGCMarkTermination                       // "GC mark termination"
	WaitReasonStoppingTheWorld                        // "stopping the world"
	WaitReasonFlushProcCaches                         // "flushing proc caches"
	WaitReasonTraceGoroutineStatus                    // "trace goroutine status"
	WaitReasonTraceProcStatus                         // "trace proc status"
	WaitReasonPageTraceFlush                          // "page trace flush"
	WaitReasonCoroutine                               // "coroutine"
	WaitReasonGCWeakToStrongWait                      // "GC weak to strong wait"
	WaitReasonSynctestRun                             // "synctest.Run"
	WaitReasonSynctestWait                            // "synctest.Wait"
	WaitReasonSynctestChanReceive                     // "chan receive (durable)"
	WaitReasonSynctestChanSend                        // "chan send (durable)"
	WaitReasonSynctestSelect                          // "select (durable)"
	WaitReasonSynctestWaitGroupWait                   // "sync.WaitGroup.Wait (durable)"
	WaitReasonCleanupWait                             // "cleanup wait"
)

var WaitReasonStrings = [...]string{
	WaitReasonZero:                  "",
	WaitReasonGCAssistMarking:       "GC assist marking",
	WaitReasonIOWait:                "IO wait",
	WaitReasonChanReceiveNilChan:    "chan receive (nil chan)",
	WaitReasonChanSendNilChan:       "chan send (nil chan)",
	WaitReasonDumpingHeap:           "dumping heap",
	WaitReasonGarbageCollection:     "garbage collection",
	WaitReasonGarbageCollectionScan: "garbage collection scan",
	WaitReasonPanicWait:             "panicwait",
	WaitReasonSelect:                "select",
	WaitReasonSelectNoCases:         "select (no cases)",
	WaitReasonGCAssistWait:          "GC assist wait",
	WaitReasonGCSweepWait:           "GC sweep wait",
	WaitReasonGCScavengeWait:        "GC scavenge wait",
	WaitReasonChanReceive:           "chan receive",
	WaitReasonChanSend:              "chan send",
	WaitReasonFinalizerWait:         "finalizer wait",
	WaitReasonForceGCIdle:           "force gc (idle)",
	WaitReasonUpdateGOMAXPROCSIdle:  "GOMAXPROCS updater (idle)",
	WaitReasonSemacquire:            "semacquire",
	WaitReasonSleep:                 "sleep",
	WaitReasonSyncCondWait:          "sync.Cond.Wait",
	WaitReasonSyncMutexLock:         "sync.Mutex.Lock",
	WaitReasonSyncRWMutexRLock:      "sync.RWMutex.RLock",
	WaitReasonSyncRWMutexLock:       "sync.RWMutex.Lock",
	WaitReasonSyncWaitGroupWait:     "sync.WaitGroup.Wait",
	WaitReasonTraceReaderBlocked:    "trace reader (blocked)",
	WaitReasonWaitForGCCycle:        "wait for GC cycle",
	WaitReasonGCWorkerIdle:          "GC worker (idle)",
	WaitReasonGCWorkerActive:        "GC worker (active)",
	WaitReasonPreempted:             "preempted",
	WaitReasonDebugCall:             "debug call",
	WaitReasonGCMarkTermination:     "GC mark termination",
	WaitReasonStoppingTheWorld:      "stopping the world",
	WaitReasonFlushProcCaches:       "flushing proc caches",
	WaitReasonTraceGoroutineStatus:  "trace goroutine status",
	WaitReasonTraceProcStatus:       "trace proc status",
	WaitReasonPageTraceFlush:        "page trace flush",
	WaitReasonCoroutine:             "coroutine",
	WaitReasonGCWeakToStrongWait:    "GC weak to strong wait",
	WaitReasonSynctestRun:           "synctest.Run",
	WaitReasonSynctestWait:          "synctest.Wait",
	WaitReasonSynctestChanReceive:   "chan receive (durable)",
	WaitReasonSynctestChanSend:      "chan send (durable)",
	WaitReasonSynctestSelect:        "select (durable)",
	WaitReasonSynctestWaitGroupWait: "sync.WaitGroup.Wait (durable)",
	WaitReasonCleanupWait:           "cleanup wait",
}

func (w WaitReason) String() string {
	if w >= WaitReason(len(WaitReasonStrings)) {
		return "unknown wait reason"
	}
	return WaitReasonStrings[w]
}
