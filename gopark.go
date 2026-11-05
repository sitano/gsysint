package gsysint

import (
	"sync/atomic"
	"unsafe"

	"github.com/sitano/gsysint/g"
	"github.com/sitano/gsysint/trace"
)

type Park struct {
	g unsafe.Pointer
}

func NewPark() Park {
	return Park{}
}

func (p *Park) Set() {
	atomic.StorePointer(&p.g, g.GetG())
}

func (p *Park) Ptr() unsafe.Pointer {
	return atomic.LoadPointer(&p.g)
}

// Park puts the current goroutine into a waiting state while holding the lock.
func (p *Park) Park(m *g.Mutex) {
	GoPark(keepLock, unsafe.Pointer(m), g.WaitReasonZero, trace.BlockGeneric, 1)
}

// keepLock is an unlockf that parks without releasing the lock (runs on g0, see ParkUnlock_c).
//
//go:norace
//go:nosplit
func keepLock(*g.G, unsafe.Pointer) bool { return true }

// GoParkUnlock puts the current goroutine into a waiting state and unlocks the lock.
// The goroutine can be made runnable again by calling goready(gp).
func (p *Park) ParkUnlock(m *g.Mutex) {
	GoParkUnlock(m, g.WaitReasonZero, trace.BlockGeneric, 1)
}

// GoReady marks gp is ready to run.
func (p *Park) Ready() {
	GoReady((*g.G)(p.Ptr()), 1)
}

// GoPark puts the current goroutine into a waiting state and calls unlockf.
// If unlockf returns false, the goroutine is resumed.
// unlockf must not access this G's stack, as it may be moved between
// the call to gopark and the call to unlockf.
// Reason explains why the goroutine has been parked.
// It is displayed in stack traces and heap dumps.
// Reasons should be unique and descriptive.
// Do not re-use reasons, add new ones.
// Lock is g.Mutex spin mutex.
//
//go:linkname GoPark runtime.gopark
func GoPark(unlockf func(*g.G, unsafe.Pointer) bool, lock unsafe.Pointer, reason g.WaitReason, traceReason trace.BlockReason, traceskip int)

// ParkUnlock_c unlocks the lock and returns true; it is an unlockf for GoPark.
// runtime.parkunlock_c is not linkname-able since go1.23, so it is reimplemented.
// It runs on g0, which has no race context: norace keeps -race from instrumenting it.
//
//go:norace
//go:nosplit
func ParkUnlock_c(gp *g.G, lock unsafe.Pointer) bool {
	g.Unlock((*g.Mutex)(lock))
	return true
}

// GoParkUnlock puts the current goroutine into a waiting state and unlocks the lock.
// The goroutine can be made runnable again by calling goready(gp).
// runtime.goparkunlock is not linkname-able since go1.23, so it is reimplemented.
func GoParkUnlock(lock *g.Mutex, reason g.WaitReason, traceReason trace.BlockReason, traceskip int) {
	GoPark(ParkUnlock_c, unsafe.Pointer(lock), reason, traceReason, traceskip)
}

// GoReady marks gp is ready to run.
// Implementation:
//
//	systemstack(func() {
//	  ready(gp, traceskip, true)
//	})
//
//go:linkname GoReady runtime.goready
func GoReady(gp *g.G, traceskip int)
