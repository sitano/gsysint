package gsysint

import (
	_ "unsafe"

	"github.com/sitano/gsysint/g"
)

// Mutual exclusion locks.  In the uncontended case,
// as fast as spin locks (just a few user-level instructions),
// but on the contention path they sleep in the kernel.
// A zeroed Mutex is unlocked (no need to initialize each lock).
type Mutex = g.Mutex

func Lock(l *Mutex) {
	g.Lock(l)
}

func Unlock(l *Mutex) {
	g.Unlock(l)
}
