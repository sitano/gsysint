package gsysint

import (
	"unsafe"
)

// Event types in the trace, args are given in square brackets.
const (
	TraceEvNone           = 0  // unused
	TraceEvBatch          = 1  // start of per-P batch of events [pid, timestamp]
	TraceEvFrequency      = 2  // contains tracer timer frequency [frequency (ticks per second)]
	TraceEvStack          = 3  // stack [stack id, number of PCs, array of PCs]
	TraceEvGomaxprocs     = 4  // current value of GOMAXPROCS [timestamp, GOMAXPROCS, stack id]
	TraceEvProcStart      = 5  // start of P [timestamp, thread id]
	TraceEvProcStop       = 6  // stop of P [timestamp]
	TraceEvGCStart        = 7  // GC start [timestamp, stack id]
	TraceEvGCDone         = 8  // GC done [timestamp]
	TraceEvGCScanStart    = 9  // GC scan start [timestamp]
	TraceEvGCScanDone     = 10 // GC scan done [timestamp]
	TraceEvGCSweepStart   = 11 // GC sweep start [timestamp, stack id]
	TraceEvGCSweepDone    = 12 // GC sweep done [timestamp]
	TraceEvGoCreate       = 13 // goroutine creation [timestamp, new goroutine id, start PC, stack id]
	TraceEvGoStart        = 14 // goroutine starts running [timestamp, goroutine id]
	TraceEvGoEnd          = 15 // goroutine ends [timestamp]
	TraceEvGoStop         = 16 // goroutine stops (like in select{}) [timestamp, stack]
	TraceEvGoSched        = 17 // goroutine calls Gosched [timestamp, stack]
	TraceEvGoPreempt      = 18 // goroutine is preempted [timestamp, stack]
	TraceEvGoSleep        = 19 // goroutine calls Sleep [timestamp, stack]
	TraceEvGoBlock        = 20 // goroutine blocks [timestamp, stack]
	TraceEvGoUnblock      = 21 // goroutine is unblocked [timestamp, goroutine id, stack]
	TraceEvGoBlockSend    = 22 // goroutine blocks on chan send [timestamp, stack]
	TraceEvGoBlockRecv    = 23 // goroutine blocks on chan recv [timestamp, stack]
	TraceEvGoBlockSelect  = 24 // goroutine blocks on select [timestamp, stack]
	TraceEvGoBlockSync    = 25 // goroutine blocks on Mutex/RWMutex [timestamp, stack]
	TraceEvGoBlockCond    = 26 // goroutine blocks on Cond [timestamp, stack]
	TraceEvGoBlockNet     = 27 // goroutine blocks on network [timestamp, stack]
	TraceEvGoSysCall      = 28 // syscall enter [timestamp, stack]
	TraceEvGoSysExit      = 29 // syscall exit [timestamp, goroutine id, real timestamp]
	TraceEvGoSysBlock     = 30 // syscall blocks [timestamp]
	TraceEvGoWaiting      = 31 // denotes that goroutine is blocked when tracing starts [goroutine id]
	TraceEvGoInSyscall    = 32 // denotes that goroutine is in syscall when tracing starts [goroutine id]
	TraceEvHeapAlloc      = 33 // memstats.heap_live change [timestamp, heap_alloc]
	TraceEvNextGC         = 34 // memstats.next_gc change [timestamp, next_gc]
	TraceEvTimerGoroutine = 35 // denotes timer goroutine [timer goroutine id]
	TraceEvFutileWakeup   = 36 // denotes that the previous wakeup of this goroutine was futile [timestamp]
	TraceEvCount          = 37
)

const (
// Timestamps in trace are cputicks/traceTickDiv.
// This makes absolute values of timestamp diffs smaller,
// and so they are encoded in less number of bytes.
// 64 on x86 is somewhat arbitrary (one tick is ~20ns on a 3GHz machine).
// The suggested increment frequency for PowerPC's time base register is
// 512 MHz according to Power ISA v2.07 section 6.2, so we use 16 on ppc64
// and ppc64le.
// Tracing won't work reliably for architectures where cputicks is emulated
// by nanotime, so the value doesn't matter for those architectures.
	TraceTickDiv = 16 + 48
// Maximum number of PCs in a single stack trace.
// Since events contain only stack id rather than whole stack trace,
// we can allow quite large values here.
	TraceStackSize = 128
// Identifier of a fake P that is used when we trace without a real P.
	TraceGlobProc = -1
// Maximum number of bytes to encode uint64 in base-128.
	TraceBytesPerNumber = 10
// Shift of the number of arguments in the first event byte.
	TraceArgCountShift = 6
// Flag passed to traceGoPark to denote that the previous wakeup of this
// goroutine was futile. For example, a goroutine was unblocked on a mutex,
// but another goroutine got ahead and acquired the mutex before the first
// goroutine is scheduled, so the first goroutine has to block again.
// Such wakeups happen on buffered channels and sync.Mutex,
// but are generally not interesting for end user.
	TraceFutileWakeup byte = 128
)

// Puts the current goroutine into a waiting state and calls unlockf.
// If unlockf returns false, the goroutine is resumed.
//go:linkname GoPark runtime.gopark
func GoPark(unlockf func(*G, unsafe.Pointer) bool, lock unsafe.Pointer, reason string, traceEv byte, traceskip int)

// Puts the current goroutine into a waiting state and unlocks the lock.
// The goroutine can be made runnable again by calling goready(gp).
//go:linkname ParkUnlock_c runtime.parkunlock_c
func ParkUnlock_c(gp *G, lock unsafe.Pointer) bool

// Puts the current goroutine into a waiting state and unlocks the lock.
// The goroutine can be made runnable again by calling goready(gp).
//go:linkname GoParkUnlock runtime.goparkunlock
func GoParkUnlock(lock *Mutex, reason string, traceEv byte, traceskip int)

//go:linkname GoReady runtime.goready
func GoReady(gp *G, traceskip int)