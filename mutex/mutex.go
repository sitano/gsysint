// Package mutex exposes the Go runtime's internal lock (runtime.mutex).
package mutex

import "github.com/sitano/gsysint/runtime/g"

// Mutex is the runtime's internal lock. In the uncontended case it is
// as fast as a spin lock (just a few user-level instructions),
// but on the contention path it sleeps in the kernel.
// A zero Mutex is unlocked.
type Mutex struct {
	m g.Mutex // must stay the first field: park passes *Mutex to the runtime as *runtime.mutex
}

func (l *Mutex) Lock() { g.Lock(&l.m) }

func (l *Mutex) Unlock() { g.Unlock(&l.m) }
