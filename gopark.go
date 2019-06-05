package gsysint

import (
	"sync/atomic"
	"unsafe"

	"github.com/sitano/gsysint/trace"

	"github.com/sitano/gsysint/g"
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

// GoPark puts the current goroutine into a waiting state and calls unlockf.
func (p *Park) Park(m *g.Mutex) {
	GoPark(func(g *g.G, p unsafe.Pointer) bool { return true }, unsafe.Pointer(m), g.WaitReasonZero, trace.TraceEvNone, 1)
}

// GoParkUnlock puts the current goroutine into a waiting state and unlocks the lock.
// The goroutine can be made runnable again by calling goready(gp).
func (p *Park) ParkUnlock(m *g.Mutex) {
	GoParkUnlock(m, g.WaitReasonZero, trace.TraceEvNone, 1)
}

// GoReady marks gp is ready to run.
func (p *Park) Ready() {
	GoReady((*g.G)(p.g), 1)
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
//go:linkname GoPark runtime.gopark
func GoPark(unlockf func(*g.G, unsafe.Pointer) bool, lock unsafe.Pointer, reason g.WaitReason, traceEv byte, traceskip int)

// ParkUnlock_c puts the current goroutine into a waiting state and unlocks the lock.
// The goroutine can be made runnable again by calling goready(gp).

//go:linkname ParkUnlock_c runtime.parkunlock_c
func ParkUnlock_c(gp *g.G, lock unsafe.Pointer) bool

// GoParkUnlock puts the current goroutine into a waiting state and unlocks the lock.
// The goroutine can be made runnable again by calling goready(gp).
//
// Implementation:
// - gopark(parkunlock_c, unsafe.Pointer(lock), reason, traceEv, traceskip)
//
//go:linkname GoParkUnlock runtime.goparkunlock
func GoParkUnlock(lock *g.Mutex, reason g.WaitReason, traceEv byte, traceskip int)

// GoReady marks gp is ready to run.
// Implementation:
// systemstack(func() {
//   ready(gp, traceskip, true)
// })
//go:linkname GoReady runtime.goready
func GoReady(gp *g.G, traceskip int)
