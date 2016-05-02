package gsysint

import (
	"unsafe"
)

/*
 * defined constants
 */
const (
	// G status
	//
	// If you add to this list, add to the list
	// of "okay during garbage collection" status
	// in mgcmark.go too.
	_Gidle            = iota // 0
	_Grunnable               // 1 runnable and on a run queue
	_Grunning                // 2
	_Gsyscall                // 3
	_Gwaiting                // 4
	_Gmoribund_unused        // 5 currently unused, but hardcoded in gdb scripts
	_Gdead                   // 6
	_Genqueue                // 7 Only the Gscanenqueue is used.
	_Gcopystack              // 8 in this state when newstack is moving the stack
	// the following encode that the GC is scanning the stack and what to do when it is done
	_Gscan = 0x1000 // atomicstatus&~Gscan = the non-scan state,
	// _Gscanidle =     _Gscan + _Gidle,      // Not used. Gidle only used with newly malloced gs
	_Gscanrunnable = _Gscan + _Grunnable //  0x1001 When scanning completes make Grunnable (it is already on run queue)
	_Gscanrunning  = _Gscan + _Grunning  //  0x1002 Used to tell preemption newstack routine to scan preempted stack.
	_Gscansyscall  = _Gscan + _Gsyscall  //  0x1003 When scanning completes make it Gsyscall
	_Gscanwaiting  = _Gscan + _Gwaiting  //  0x1004 When scanning completes make it Gwaiting
// _Gscanmoribund_unused,               //  not possible
// _Gscandead,                          //  not possible
	_Gscanenqueue = _Gscan + _Genqueue //  When scanning completes make it Grunnable and put on runqueue
)

const (
	// P status
	_Pidle    = iota
	_Prunning // Only this P is allowed to change from _Prunning.
	_Psyscall
	_Pgcstop
	_Pdead
)

type Mutex struct {
	// Futex-based impl treats it as uint32 key,
	// while sema-based impl as M* waitm.
	// Used to be a union, but unions break precise GC.
	key uintptr
}

type Note struct {
	// Futex-based impl treats it as uint32 key,
	// while sema-based impl as M* waitm.
	// Used to be a union, but unions break precise GC.
	key uintptr
}

type FuncVal struct {
	fn uintptr
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
// Gs, Ms, and Ps are always reachable via true pointers in the
// allgs, allm, and allp lists or (during allocation before they reach those lists)
// from stack variables.

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
func (gp Guintptr) ptr() *G { return (*G)(unsafe.Pointer(gp)) }

//go:nosplit
func (gp *Guintptr) set(g *G) { *gp = Guintptr(unsafe.Pointer(g)) }

//go:nosplit
//func (gp *guintptr) cas(old, new guintptr) bool {
//	return atomic.Casuintptr((*uintptr)(unsafe.Pointer(gp)), uintptr(old), uintptr(new))
//}

type Puintptr uintptr

type Muintptr uintptr

//go:nosplit
func (mp Muintptr) ptr() *M { return (*M)(unsafe.Pointer(mp)) }

//go:nosplit
func (mp *Muintptr) set(m *M) { *mp = Muintptr(unsafe.Pointer(m)) }

type Uintreg uint64

type GoBuf struct {
	// The offsets of sp, pc, and g are known to (hard-coded in) libmach.
	sp   uintptr
	pc   uintptr
	g    Guintptr
	ctxt unsafe.Pointer // this has to be a pointer so that gc scans it
	ret  Uintreg
	lr   uintptr
	bp   uintptr // for GOEXPERIMENT=framepointer
}

// Known to compiler.
// Changes here must also be made in src/cmd/internal/gc/select.go's selecttype.
type Sudog struct {
	g           *G
	selectdone  *uint32
	next        *Sudog
	prev        *Sudog
	elem        unsafe.Pointer // data element
	releasetime int64
	nrelease    int32          // -1 for acquire
	waitlink    *Sudog         // g.waiting list
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
	_panic  *Panic  // panic that is running defer
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
	stack       Stack           // offset known to runtime/cgo
	stackguard0 uintptr         // offset known to liblink
	stackguard1 uintptr         // offset known to liblink

	_panic      *Panic          // innermost panic - offset known to liblink
	_defer      *Defer          // innermost defer
	m           *M              // current m; offset known to arm liblink
	stackAlloc  uintptr         // stack allocation is [stack.lo,stack.lo+stackAlloc)
	sched       GoBuf
	syscallsp   uintptr         // if status==Gsyscall, syscallsp = sched.sp to use during gc
	syscallpc    uintptr        // if status==Gsyscall, syscallpc = sched.pc to use during gc
	stkbar       []StkBar       // stack barriers, from low to high (see top of mstkbar.go)
	stkbarPos    uintptr        // index of lowest stack barrier not hit
	stktopsp     uintptr        // expected sp at top of stack, to check in traceback
	param        unsafe.Pointer // passed parameter on wakeup
	atomicstatus uint32
	stackLock      uint32       // sigprof/scang lock; TODO: fold in to atomicstatus
	goid           int64
	waitsince      int64        // approx time when the g become blocked
	waitreason     string       // if status==Gwaiting
	schedlink      Guintptr
	preempt        bool         // preemption signal, duplicates stackguard0 = stackpreempt
	paniconfault   bool         // panic (instead of crash) on unexpected fault address
	preemptscan    bool         // preempted g does scan for gc
	gcscandone     bool         // g has scanned stack; protected by _Gscan bit in status
	gcscanvalid    bool         // false at start of gc cycle, true if G has not run since last scan
	throwsplit     bool         // must not split stack
	raceignore     int8         // ignore race detection events
	sysblocktraced bool         // StartTrace has emitted EvGoInSyscall about this goroutine
	sysexitticks   int64        // cputicks when syscall has returned (for tracing)
	sysexitseq     uint64       // trace seq when syscall has returned (for tracing)
	lockedm        *M
	sig            uint32
	writebuf       []byte
	sigcode0       uintptr
	sigcode1       uintptr
	sigpc          uintptr
	gopc           uintptr      // pc of go statement that created this goroutine
	startpc        uintptr      // pc of goroutine function
	racectx        uintptr
	waiting        *Sudog       // sudog structures this g is waiting on (that have a valid elem ptr)

								// Per-G gcController state

								// gcAssistBytes is this G's GC assist credit in terms of
								// bytes allocated. If this is positive, then the G has credit
								// to allocate gcAssistBytes bytes without assisting. If this
								// is negative, then the G must correct this by performing
								// scan work. We track this in bytes to make it fast to update
								// and check for debt in the malloc hot path. The assist ratio
								// determines how this corresponds to scan work debt.
	gcAssistBytes int64
}

type M struct {
	g0      *G                   // goroutine with scheduling stack
	morebuf GoBuf                // gobuf arg to morestack
	divmod  uint32               // div/mod denominator for arm - known to liblink
								 // Fields not known to debuggers.
	procid        uint64         // for debuggers, but offset not hard-coded
	gsignal       *G             // signal-handling g
	sigmask       SigSet         // storage for saved signal mask
	tls           [6]uintptr     // thread-local storage (for x86 extern register)
	mstartfn      func()
	curg          *G             // current running goroutine
	caughtsig     Guintptr       // goroutine running during fatal signal
	p             Puintptr       // attached p for executing go code (nil if not executing go code)
	nextp         Puintptr
	id            int32
	mallocing     int32
	throwing      int32
	preemptoff    string         // if != "", keep curg running on this m
	locks         int32
	softfloat     int32
	dying         int32
	profilehz     int32
	helpgc        int32
	spinning      bool           // m is out of work and is actively looking for work
	blocked       bool           // m is blocked on a note
	inwb          bool           // m is executing a write barrier
	newSigstack   bool           // minit on C thread called sigaltstack
	printlock     int8
	fastrand      uint32
	ncgocall      uint64         // number of cgo calls in total
	ncgo          int32          // number of cgo calls currently in progress
	park          Note
	alllink       *M             // on allm
	schedlink     Muintptr
	machport      uint32         // return address for mach ipc (os x)
	mcache        *MCache
	lockedg       *G
	createstack   [32]uintptr    // stack that created this thread.
	freglo        [16]uint32     // d[i] lsb and f[i]
	freghi        [16]uint32     // d[i] msb and f[i+16]
	fflag         uint32         // floating point compare flags
	locked        uint32         // tracking for lockosthread
	nextwaitm     uintptr        // next m waiting for lock
	gcstats       GCStats
	needextram    bool
	traceback     uint8
	waitunlockf   unsafe.Pointer // todo go func(*g, unsafe.pointer) bool
	waitlock      unsafe.Pointer
	waittraceev   byte
	waittraceskip int
	startingtrace bool
	syscalltick   uint32
								 //#ifdef GOOS_windows
	thread uintptr               // thread handle
								 // these are here because they are too large to be on the stack
								 // of low-level NOSPLIT functions.
	libcall   LibCall
	libcallpc uintptr            // for cpu profiler
	libcallsp uintptr
	libcallg  Guintptr
	syscall   LibCall            // stores syscall parameters on windows
								 //#endif
	mOS
}
