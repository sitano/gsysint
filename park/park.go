// Package park parks and readies goroutines via the runtime scheduler (gopark/goready).
package park

import (
	"sync/atomic"
	"unsafe"

	"github.com/sitano/gsysint/internal/race"
	"github.com/sitano/gsysint/mutex"
	"github.com/sitano/gsysint/runtime/g"
	"github.com/sitano/gsysint/runtime/trace"
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
func (p *Park) Park(m *mutex.Mutex) {
	GoPark(KeepLock, unsafe.Pointer(m), g.WaitReasonZero, trace.BlockGeneric, 1)
}

// KeepLock is an unlockf that parks without releasing the lock (runs on g0, see ParkUnlock_c).
//
//go:norace
//go:nosplit
func KeepLock(*g.G, unsafe.Pointer) bool { return true }

// ParkUnlock puts the current goroutine into a waiting state and unlocks the lock.
// The goroutine can be made runnable again by calling Ready.
func (p *Park) ParkUnlock(m *mutex.Mutex) {
	GoParkUnlock(m, g.WaitReasonZero, trace.BlockGeneric, 1)
}

// IsParked reports whether the goroutine recorded by Set is parked (see IsParked).
func (p *Park) IsParked() bool {
	gp := (*g.G)(p.Ptr())
	return gp != nil && IsParked(gp)
}

// Ready marks the parked goroutine ready to run.
// It throws "bad g->status in ready" if the goroutine is not parked yet:
// use IsParked, a handshake, or event.Event, which has no such ordering hazard.
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
// Lock is a *mutex.Mutex for ParkUnlock_c.
//
//go:linkname GoPark runtime.gopark
func GoPark(unlockf func(*g.G, unsafe.Pointer) bool, lock unsafe.Pointer, reason g.WaitReason, traceReason trace.BlockReason, traceskip int)

// ParkUnlock_c unlocks the lock (a *mutex.Mutex) and returns true; it is an unlockf for GoPark.
// runtime.parkunlock_c is not linkname-able since go1.23, so it is reimplemented.
// It runs on g0, which has no race context: norace keeps -race from instrumenting it.
// It calls g.Unlock directly, not (*mutex.Mutex).Unlock, which -race would instrument.
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
func GoParkUnlock(lock *mutex.Mutex, reason g.WaitReason, traceReason trace.BlockReason, traceskip int) {
	// ParkUnlock_c releases lock on g0, invisible to -race: publish the release here.
	race.Release(unsafe.Pointer(lock))
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
