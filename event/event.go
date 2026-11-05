// Package event is a one-shot, latching goroutine wakeup built on gopark:
// Set and Wait may happen in either order.
package event

import (
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/sitano/gsysint/park"
	"github.com/sitano/gsysint/runtime/g"
	"github.com/sitano/gsysint/runtime/trace"
)

// Event states. Any other value is the *g.G of the parked waiter
// (g structs are pointer-aligned, so they never collide with 1 or 2).
const (
	evEmpty   uintptr = 0 // not set, nobody parked
	evSet     uintptr = 1 // set; latched forever
	evTimeout uintptr = 2 // WaitTimeout's timer fired before the waiter parked
)

// Event is a one-shot, latching wakeup for a single waiter: the runtime's
// "note" (notesleep/notewakeup) rebuilt on gopark instead of a futex.
//
// Unlike park.Park, Set and Wait can happen in either order: Set before Wait makes
// Wait return immediately, and Set never readies a goroutine that is not parked
// yet. The whole state is one word, changed only by CAS/swap:
//
//	evEmpty --Wait parks--> gp --Set-->     evSet  (Set readies gp)
//	evEmpty --Set-------------------------> evSet  (Wait won't park)
//	gp      --timer------------------------> evEmpty (timer readies gp, Wait returns false)
//	evEmpty --timer, before park-----------> evTimeout (park is cancelled)
//
// The race between "Wait decides to park" and "Set fires" is closed inside
// gopark's unlockf (eventPark), which runs on g0 after the waiter is already
// _Gwaiting: it publishes gp with a CAS from evEmpty. If the CAS fails, the
// event was set (or timed out) in between, and unlockf returns false, which
// makes gopark resume the waiter at once instead of scheduling away. So
// whoever reads gp out of the state word knows it is _Gwaiting and may GoReady it.
//
// Only one goroutine may wait at a time; a second concurrent waiter panics.
// Any number of goroutines may call Set. There is no Reset: once set, an Event
// stays set. The zero Event is ready to use. It must not be copied after first use.
//
// Set happens-before every Wait/WaitTimeout that returns true (the state word is
// accessed with sync/atomic on the goroutine side, which -race understands).
type Event struct {
	state uintptr
}

// Set sets the event and readies the waiter, if one is parked. Idempotent.
func (e *Event) Set() {
	if s := atomic.SwapUintptr(&e.state, evSet); s > evTimeout {
		park.GoReady(gptr(s), 1)
	}
}

// IsSet reports whether Set has been called.
func (e *Event) IsSet() bool { return atomic.LoadUintptr(&e.state) == evSet }

// Wait blocks until the event is set.
func (e *Event) Wait() {
	if e.IsSet() {
		return
	}
	park.GoPark(eventPark, unsafe.Pointer(e), g.WaitReasonSyncCondWait, trace.BlockCondWait, 1)
	if !e.IsSet() {
		panic("event: Event has another waiter")
	}
}

// WaitTimeout blocks until the event is set or d elapses, and reports whether
// it was set. After a timeout the Event is still usable: wait again or Set it.
func (e *Event) WaitTimeout(d time.Duration) bool {
	if e.IsSet() {
		return true
	}
	t := time.AfterFunc(d, e.timeout)
	park.GoPark(eventPark, unsafe.Pointer(e), g.WaitReasonSyncCondWait, trace.BlockCondWait, 1)
	t.Stop()
	switch s := atomic.LoadUintptr(&e.state); s {
	case evSet:
		return true
	case evEmpty: // the timer readied us
		return false
	case evTimeout: // the timer fired before we parked; undo it, unless Set beat us
		return !atomic.CompareAndSwapUintptr(&e.state, evTimeout, evEmpty)
	default:
		panic("event: Event has another waiter")
	}
}

// timeout runs on the timer goroutine of WaitTimeout. It either takes the
// parked waiter out of the state word and readies it, or, if the waiter has not
// parked yet, leaves evTimeout so that its eventPark fails and it never parks.
// A late timer (the wait already returned because of Set) sees evSet and does nothing.
func (e *Event) timeout() {
	for {
		switch s := atomic.LoadUintptr(&e.state); s {
		case evSet, evTimeout:
			return
		case evEmpty:
			if atomic.CompareAndSwapUintptr(&e.state, evEmpty, evTimeout) {
				return
			}
		default:
			if atomic.CompareAndSwapUintptr(&e.state, s, evEmpty) {
				park.GoReady(gptr(s), 1)
				return
			}
		}
	}
}

// eventPark is Event's unlockf. It runs on g0 with the waiter already
// _Gwaiting, so it can neither use sync/atomic (instrumented under -race, and g0
// has no race context) nor touch the waiter's stack. cas is plain assembly.
//
//go:norace
//go:nosplit
func eventPark(gp *g.G, e unsafe.Pointer) bool {
	return cas(&(*Event)(e).state, evEmpty, uintptr(unsafe.Pointer(gp)))
}

// gptr turns a state word back into the waiter's g. Safe because the runtime
// never frees g structs (allgs keeps them alive), so the uintptr cannot dangle.
func gptr(s uintptr) *g.G {
	return *(**g.G)(unsafe.Pointer(&s))
}

// cas is a sequentially consistent compare-and-swap with no race
// instrumentation and no stack growth, callable from g0 (cas_*.s).
//
//go:noescape
func cas(p *uintptr, old, new uintptr) bool
