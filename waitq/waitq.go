// Package waitq is a runtime-level condition variable: a FIFO of goroutines
// parked with gopark and guarded by a mutex.Mutex.
package waitq

import (
	"github.com/sitano/gsysint/mutex"
	"github.com/sitano/gsysint/park"
	"github.com/sitano/gsysint/runtime/g"
	"github.com/sitano/gsysint/runtime/trace"
)

// Queue is a FIFO of parked goroutines guarded by an external mutex.Mutex:
// a runtime-level sync.Cond built from park.GoParkUnlock and park.GoReady only.
//
// Every method must be called with the guarding mutex held. That is what
// makes it lose no wakeups:
//
//   - Wait appends the current g while holding m and parks with
//     park.GoParkUnlock, which releases m on g0 only after the g is _Gwaiting.
//   - Signal/Broadcast need m, so they can only see a queued g once it is
//     already _Gwaiting, and GoReady(gp) is always legal for it.
//   - A g is removed from the queue before it is readied, so it is readied
//     exactly once and the queue never holds a goroutine that has moved on.
//
// Usage, the classic condition loop:
//
//	m.Lock()
//	for !cond {
//		q.Wait(&m)
//	}
//	// cond holds, m is held
//	m.Unlock()
//
//	m.Lock()
//	cond = true
//	q.Signal()
//	m.Unlock()
//
// Like any user of mutex.Mutex, code between Lock and Unlock must not block.
// The zero Queue is empty and ready to use. It must not be copied after first use.
type Queue struct {
	gs []*g.G
}

// Wait parks the calling goroutine until Signal or Broadcast wakes it.
// m must be held; it is released while parked and held again when Wait returns.
// As with sync.Cond, re-check the condition after Wait returns.
func (q *Queue) Wait(m *mutex.Mutex) {
	q.gs = append(q.gs, g.CurG())
	park.GoParkUnlock(m, g.WaitReasonSyncCondWait, trace.BlockCondWait, 1)
	m.Lock()
}

// Signal readies the longest-waiting goroutine, if any, and reports whether it did.
// The readied goroutine runs once it reacquires m, i.e. after the caller unlocks.
func (q *Queue) Signal() bool {
	if len(q.gs) == 0 {
		return false
	}
	gp := q.gs[0]
	q.gs[0] = nil
	q.gs = q.gs[1:]
	park.GoReady(gp, 1)
	return true
}

// Broadcast readies all waiting goroutines in FIFO order and returns how many.
func (q *Queue) Broadcast() int {
	n := len(q.gs)
	for i, gp := range q.gs {
		q.gs[i] = nil
		park.GoReady(gp, 1)
	}
	q.gs = q.gs[:0]
	return n
}

// Len returns the number of parked goroutines.
func (q *Queue) Len() int { return len(q.gs) }
