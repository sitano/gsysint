// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package g

import (
	"unsafe"

	"github.com/sitano/gsysint/sys"
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
	_Gscan         = 0x1000
	_Gscanrunnable = _Gscan + _Grunnable // 0x1001
	_Gscanrunning  = _Gscan + _Grunning  // 0x1002
	_Gscansyscall  = _Gscan + _Gsyscall  // 0x1003
	_Gscanwaiting  = _Gscan + _Gwaiting  // 0x1004
)

const (
	// P status
	_Pidle    = iota
	_Prunning // Only this P is allowed to change from _Prunning.
	_Psyscall
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

//go:nosplit
//func (gp *guintptr) Cas(old, new guintptr) bool {
//	return atomic.Casuintptr((*uintptr)(unsafe.Pointer(gp)), uintptr(old), uintptr(new))
//}

type Puintptr uintptr

//go:nosplit
//func (pp Puintptr) ptr() *P { return (*P)(unsafe.Pointer(pp)) }

//go:nosplit
//func (pp *Puintptr) set(p *P) { *pp = Puintptr(unsafe.Pointer(p)) }

// muintptr is a *m that is not tracked by the garbage collector.
//
// Because we do free Ms, there are some additional constrains on
// muintptrs:
//
// 1. Never hold an muintptr locally across a safe point.
//
// 2. Any muintptr in the heap must be owned by the M itself so it can
//    ensure it is not in use when the last true *m is released.
type Muintptr uintptr

//go:nosplit
func (mp Muintptr) Ptr() *M { return (*M)(unsafe.Pointer(mp)) }

//go:nosplit
func (mp *Muintptr) Set(m *M) { *mp = Muintptr(unsafe.Pointer(m)) }

type Uintreg uint64

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
	ret  sys.Uintreg
	lr   uintptr
	bp   uintptr // for GOEXPERIMENT=framepointer
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
	// the following fields are protected by the hchan.lock of the
	// channel this sudog is blocking on. shrinkstack depends on
	// this for sudogs involved in channel ops.

	g *G

	// isselect indicates g is participating in a select, so
	// g.selectdone must be cas'd to win the wake-up race.
	isselect bool
	next     *Sudog
	prev     *Sudog
	elem     unsafe.Pointer // data element (may point to stack)

	// the following fields are never accessed concurrently.
	// for channels, waitlink is only accessed by g.
	// for semaphores, all fields (including the ones above)
	// are only accessed when holding a semaroot lock.

	acquiretime int64
	releasetime int64
	ticket      uint32
	parent      *Sudog // semaroot binary tree
	waitlink    *Sudog // g.waiting list or semaroot
	waittail    *Sudog // semaroot
	c           *HChan // channel
}

type GCStats struct {
	// the struct must consist of only uint64's,
	// because it is casted to uint64[].
	nhandoff    uint64
	nhandoffcnt uint64
	nprocyield  uint64
	nosyield    uint64
	nsleep      uint64
}

type LibCall struct {
	fn   uintptr
	n    uintptr // number of parameters
	args uintptr // parameters
	r1   uintptr // return values
	r2   uintptr
	err  uintptr // error number
}

// describes how to handle callback
type WinCallBackContext struct {
	gobody       unsafe.Pointer // go function to call
	argsize      uintptr        // callback arguments size (in bytes)
	restorestack uintptr        // adjust stack on return by (in bytes) (386 only)
	cleanstack   bool
}

// Stack describes a Go execution stack.
// The bounds of the stack are exactly [lo, hi),
// with no implicit data structures on either side.
type Stack struct {
	lo uintptr
	hi uintptr
}

// stkbar records the state of a G's stack barrier.
type StkBar struct {
	savedLRPtr uintptr // location overwritten by stack barrier PC
	savedLRVal uintptr // value overwritten at savedLRPtr
}

/*
 * deferred subroutine calls
 */
type Defer struct {
	siz     int32
	started bool
	sp      uintptr // sp at time of defer
	pc      uintptr
	fn      *FuncVal
	_panic  *Panic // panic that is running defer
	link    *Defer
}

/*
 * panics
 */
type Panic struct {
	argp      unsafe.Pointer // pointer to arguments of deferred call run during panic; cannot move - known to liblink
	arg       interface{}    // argument to panic
	link      *Panic         // link to earlier panic
	recovered bool           // whether this panic is over
	aborted   bool           // the panic was aborted
}

// Layout of in-memory per-function information prepared by linker
// See https://golang.org/s/go12symtab.
// Keep in sync with linker
// and with package debug/gosym and with symtab.go in package runtime.
type Func struct {
	entry   uintptr // start pc
	nameoff int32   // function name

	args int32 // in/out args size
	_    int32 // previously legacy frame size; kept for layout compatibility

	pcsp      int32
	pcfile    int32
	pcln      int32
	npcdata   int32
	nfuncdata int32
}

/*
 * stack traces
 */

type StkFrame struct {
	fn       *Func      // function being run
	pc       uintptr    // program counter within fn
	continpc uintptr    // program counter where execution can continue, or 0 if not
	lr       uintptr    // program counter at caller aka link register
	sp       uintptr    // stack pointer at pc
	fp       uintptr    // stack pointer at caller aka frame pointer
	varp     uintptr    // top of local variables
	argp     uintptr    // pointer to function arguments
	arglen   uintptr    // number of bytes at argp
	argmap   *BitVector // force use of this argmap
}

// AncestorInfo records details of where a goroutine was started.
type AncestorInfo struct {
	PCS  []uintptr // pcs from the stack of this goroutine
	GoID int64     // goroutine id of this goroutine; original goroutine possibly dead
	GoPC uintptr   // pc of go statement that created this goroutine
}

// Per-thread (in Go, per-P) cache for small objects.
// No locking needed because it is per-thread (per-P).
//
// mcaches are allocated from non-GC'd memory, so any heap pointers
// must be specially handled.
type MCache struct {
	// ...
}

type G struct {
	// Stack parameters.
	// stack describes the actual stack memory: [stack.lo, stack.hi).
	// stackguard0 is the stack pointer compared in the Go stack growth prologue.
	// It is stack.lo+StackGuard normally, but can be StackPreempt to trigger a preemption.
	// stackguard1 is the stack pointer compared in the C stack growth prologue.
	// It is stack.lo+StackGuard on g0 and gsignal stacks.
	// It is ~0 on other goroutine stacks, to trigger a call to morestackc (and crash).
	Stack       Stack   // offset known to runtime/cgo
	StackGuard0 uintptr // offset known to liblink
	StackGuard1 uintptr // offset known to liblink

	Panic          *Panic // innermost panic - offset known to liblink
	Defer          *Defer // innermost defer
	M              *M     // current m; offset known to arm liblink
	Sched          GoBuf
	SysCallSP      uintptr        // if status==Gsyscall, syscallsp = sched.sp to use during gc
	SysCallPC      uintptr        // if status==Gsyscall, syscallpc = sched.pc to use during gc
	StkTopSP       uintptr        // expected sp at top of stack, to check in traceback
	Param          unsafe.Pointer // passed parameter on wakeup
	AtomicStatus   uint32
	StackLock      uint32 // sigprof/scang lock; TODO: fold in to atomicstatus
	GoID           int64
	SchedLink      Guintptr
	WaitSince      int64      // approx time when the g become blocked
	WaitReason     WaitReason // if status==Gwaiting
	Preempt        bool       // preemption signal, duplicates stackguard0 = stackpreempt
	PanicOnFault   bool       // panic (instead of crash) on unexpected fault address
	PreemptScan    bool       // preempted g does scan for gc
	GcScanDone     bool       // g has scanned stack; protected by _Gscan bit in status
	GcScanValid    bool       // false at start of gc cycle, true if G has not run since last scan; TODO: remove?
	ThrowSplit     bool       // must not split stack
	RaceIgnore     int8       // ignore race detection events
	SysBlockTraced bool       // StartTrace has emitted EvGoInSyscall about this goroutine
	SysExitTicks   int64      // cputicks when syscall has returned (for tracing)
	TraceSeq       uint64     // trace event sequencer
	TraceLastP     Puintptr   // last P emitted an event for this goroutine
	LockedM        Muintptr
	Sig            uint32
	WriteBuf       []byte
	SigCode0       uintptr
	SigCode1       uintptr
	SigPC          uintptr
	GoPC           uintptr         // pc of go statement that created this goroutine
	Ancestors      *[]AncestorInfo // ancestor information goroutine(s) that created this goroutine (only used if debug.tracebackancestors)
	StartPC        uintptr         // pc of goroutine function
	RaceCtx        uintptr
	Waiting        *Sudog         // sudog structures this g is waiting on (that have a valid elem ptr); in lock order
	CgoCtxt        []uintptr      // cgo traceback context
	Labels         unsafe.Pointer // profiler labels
	Timer          *Timer         // cached timer for time.Sleep
	SelectDone     uint32         // are we participating in a select and did someone win the race?

	// Per-G GC state

	// gcAssistBytes is this G's GC assist credit in terms of
	// bytes allocated. If this is positive, then the G has credit
	// to allocate gcAssistBytes bytes without assisting. If this
	// is negative, then the G must correct this by performing
	// scan work. We track this in bytes to make it fast to update
	// and check for debt in the malloc hot path. The assist ratio
	// determines how this corresponds to scan work debt.
	GcAssistBytes int64
}

type M struct {
	G0      *G     // goroutine with scheduling stack
	MoreBuf GoBuf  // gobuf arg to morestack
	DivMod  uint32 // div/mod denominator for arm - known to liblink

	// Fields not known to debuggers.
	ProcID        uint64       // for debuggers, but offset not hard-coded
	GSignal       *G           // signal-handling g
	GoSigStack    GSignalStack // Go-allocated signal handling stack
	SigMask       SigSet       // storage for saved signal mask
	TLS           [6]uintptr   // thread-local storage (for x86 extern register)
	MStartFn      func()
	CurG          *G       // current running goroutine
	CaughtSig     Guintptr // goroutine running during fatal signal
	P             Puintptr // attached p for executing go code (nil if not executing go code)
	NextP         Puintptr
	OldP          Puintptr // the p that was attached before executing a syscall
	ID            int64
	MAllocing     int32
	Throwing      int32
	PreemptOff    string // if != "", keep curg running on this m
	Locks         int32
	Dying         int32
	ProfileHz     int32
	Spinning      bool // m is out of work and is actively looking for work
	Blocked       bool // m is blocked on a note
	InWb          bool // m is executing a write barrier
	NewSigStack   bool // minit on C thread called sigaltstack
	PrintLock     int8
	IncGo         bool   // m is executing a cgo call
	FreeWait      uint32 // if == 0, safe to free g0 and delete m (atomic)
	FastRand      [2]uint32
	NeedextRam    bool
	TraceBack     uint8
	NCgoCall      uint64      // number of cgo calls in total
	NCgo          int32       // number of cgo calls currently in progress
	CgoCallersUse uint32      // if non-zero, cgoCallers in use temporarily
	CgoCallers    *CgoCallers // cgo traceback if crashing in cgo call
	Park          Note
	AllLink       *M // on allm
	SchedLink     Muintptr
	MCache        *MCache
	LockedG       Guintptr
	CreateStack   [32]uintptr    // stack that created this thread.
	LockedExt     uint32         // tracking for external LockOSThread
	LockedInt     uint32         // tracking for internal lockOSThread
	NextWaitM     Muintptr       // next m waiting for lock
	WaitUnlockF   unsafe.Pointer // todo go func(*g, unsafe.pointer) bool
	WaitLock      unsafe.Pointer
	WaitTraceEv   byte
	WaitTraceSkip int
	StartingTrace bool
	SysCallTick   uint32
	Thread        uintptr // thread handle
	FreeLink      *M      // on sched.freem

	// these are here because they are too large to be on the stack
	// of low-level NOSPLIT functions.
	LibCall   LibCall
	LibCallPC uintptr // for cpu profiler
	LibCallSP uintptr
	LibCallG  Guintptr
	Syscall   LibCall // stores syscall parameters on windows

	VDSOSP uintptr // SP for traceback while in VDSO call (0 if not in call)
	VDSOPC uintptr // PC for traceback while in VDSO call

	MOS
}

// A waitReason explains why a goroutine has been stopped.
// See gopark. Do not re-use waitReasons, add new ones.
type WaitReason uint8

const (
	WaitReasonZero                  WaitReason = iota // ""
	WaitReasonGCAssistMarking                         // "GC assist marking"
	WaitReasonIOWait                                  // "IO wait"
	WaitReasonChanReceiveNilChan                      // "chan receive (nil chan)"
	WaitReasonChanSendNilChan                         // "chan send (nil chan)"
	WaitReasonDumpingHeap                             // "dumping heap"
	WaitReasonGarbageCollection                       // "garbage collection"
	WaitReasonGarbageCollectionScan                   // "garbage collection scan"
	WaitReasonPanicWait                               // "panicwait"
	WaitReasonSelect                                  // "select"
	WaitReasonSelectNoCases                           // "select (no cases)"
	WaitReasonGCAssistWait                            // "GC assist wait"
	WaitReasonGCSweepWait                             // "GC sweep wait"
	WaitReasonChanReceive                             // "chan receive"
	WaitReasonChanSend                                // "chan send"
	WaitReasonFinalizerWait                           // "finalizer wait"
	WaitReasonForceGGIdle                             // "force gc (idle)"
	WaitReasonSemacquire                              // "semacquire"
	WaitReasonSleep                                   // "sleep"
	WaitReasonSyncCondWait                            // "sync.Cond.Wait"
	WaitReasonTimerGoroutineIdle                      // "timer goroutine (idle)"
	WaitReasonTraceReaderBlocked                      // "trace reader (blocked)"
	WaitReasonWaitForGCCycle                          // "wait for GC cycle"
	WaitReasonGCWorkerIdle                            // "GC worker (idle)"
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
	WaitReasonChanReceive:           "chan receive",
	WaitReasonChanSend:              "chan send",
	WaitReasonFinalizerWait:         "finalizer wait",
	WaitReasonForceGGIdle:           "force gc (idle)",
	WaitReasonSemacquire:            "semacquire",
	WaitReasonSleep:                 "sleep",
	WaitReasonSyncCondWait:          "sync.Cond.Wait",
	WaitReasonTimerGoroutineIdle:    "timer goroutine (idle)",
	WaitReasonTraceReaderBlocked:    "trace reader (blocked)",
	WaitReasonWaitForGCCycle:        "wait for GC cycle",
	WaitReasonGCWorkerIdle:          "GC worker (idle)",
}

func (w WaitReason) String() string {
	if w < 0 || w >= WaitReason(len(WaitReasonStrings)) {
		return "unknown wait reason"
	}
	return WaitReasonStrings[w]
}
