// Package mutex exposes the Go runtime's internal lock (runtime.mutex).
package mutex

import (
	"unsafe"

	"github.com/sitano/gsysint/internal/race"
	"github.com/sitano/gsysint/runtime/g"
)

// Mutex is the runtime's internal lock. In the uncontended case it is
// as fast as a spin lock (just a few user-level instructions),
// but on the contention path it sleeps in the kernel.
// A zero Mutex is unlocked.
//
// The lock is owned by the M (thread), not the goroutine: holding it bumps
// m.locks, which disables preemption. A goroutine holding it must not block
// (channel op, sync.Mutex, syscall-park...), or the scheduler throws
// "schedule: holding locks". The only way to block with it held is to park
// with an unlockf that releases it (see park.GoParkUnlock, waitq.Queue).
//
// Under -race, Lock/Unlock are annotated as acquire/release so data guarded
// by the Mutex is not reported as racy.
type Mutex struct {
	m g.Mutex // must stay the first field: park passes *Mutex to the runtime as *runtime.mutex
}

func (l *Mutex) Lock() {
	g.Lock(&l.m)
	race.Acquire(unsafe.Pointer(l))
}

func (l *Mutex) Unlock() {
	race.Release(unsafe.Pointer(l))
	g.Unlock(&l.m)
}
