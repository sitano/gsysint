# gsysint

[![ci](https://github.com/sitano/gsysint/actions/workflows/ci.yml/badge.svg)](https://github.com/sitano/gsysint/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/sitano/gsysint.svg)](https://pkg.go.dev/github.com/sitano/gsysint)

Golang (as of 1.27.2, amd64/386/arm64) runtime internals that gives you an access to internal scheduling primitives.
(for learning purposes)

Features
========

* `g` and `m` internal structures access (read goroutine id)
* `park`: goroutines native parking / unparking
* `mutex`: internal spin lock
* `waitq.Queue`: a runtime-level condition variable (FIFO of parked goroutines)
* `event.Event`: a one-shot wakeup that works whatever the order of Set and Wait, with `WaitTimeout`
* `park.IsParked` / `g.(*G).Status`: inspect goroutine scheduling state

Examples
=======

Get goroutine id:

    gsysint.GoID()

    or

    gsysint.GIDFromStackTrace()

Test if goroutine parked:

    var gp = g.CurG() // from some goroutine
    park.IsParked(gp)

Park goroutine the simple way:

    var p park.Park
    var m mutex.Mutex

    go func() {
        p.Set()
        m.Lock()
        p.ParkUnlock(&m)
    }()
    
    // unpark goroutine and mark as ready
    p.Ready()
    
Park goroutine harder with mutex release on park:

    var gp atomic.Pointer[g.G]

    w := sync.WaitGroup{}
    w.Add(1)

    l := &mutex.Mutex{}
    go func() {
        gp.Store(g.CurG())
        l.Lock()
        w.Done()
        // park, l is released only after the goroutine is parked
        park.GoParkUnlock(l, g.WaitReasonZero, trace.BlockGeneric, 1)
    }()

    w.Wait()   // goroutine has started and holds l
    l.Lock()   // succeeds only once the goroutine is parked
    // unpark goroutine and mark as ready
    park.GoReady(gp.Load(), 1)
    l.Unlock()

Condition variable over the runtime mutex:

    var m mutex.Mutex
    var q waitq.Queue
    ready := false

    go func() {
        m.Lock()
        for !ready {
            q.Wait(&m) // parks, releasing m; holds m again on return
        }
        m.Unlock()
    }()

    m.Lock()
    ready = true
    q.Signal()
    m.Unlock()

One-shot event, Set and Wait in any order:

    var e event.Event
    go e.Set()
    if !e.WaitTimeout(time.Second) {
        // timed out; e is still usable
    }

Scheduling details
==================

I am not going to cover go [scheduler](
https://github.com/golang/go/blob/7bc40ffb05d8813bf9b41a331b45d37216f9e747/src/runtime/proc.go#L2022)
in [details](https://golang.org/s/go11sched) here.

The scheduler's job is to distribute ready-to-run goroutines
over worker threads. Main concepts:

- G - goroutine.
- M - worker thread, or machine.
- P - processor, a resource that is required to execute Go code.
      M must have an associated P to execute Go code, however it can be
      blocked or in a syscall w/o an associated P.

Runtime defined as a tuple of (m0, g0). Almost everything interested is happening in
the context of g0 (like scheduling, gc setup, etc). Usually switch from an arbitrary
goroutine to the g0 can happen in the case of: resceduling, goroutine parking, exiting /
finishing, syscalling, recovery from panic and maybe other cases I did not managed
to find with grep. In order to do a switch runtime calls [mcall](
https://github.com/golang/go/blob/7bc40ffb05d8813bf9b41a331b45d37216f9e747/src/runtime/stubs.go#L34)
function.

`mcall` switches from the g to the g0 stack and invokes fn(g), where g is the
goroutine that made the call. mcall can only be called from g stacks (not g0, not gsignal).

Parking
=======

A goroutine can be took off the scheduling for a while to keep resources free until
some condition met. It called `parking`. Often and almost always go runtime uses
this method to implement various synchronisation primitives behavior and implementation.

* `gopark` puts the current goroutine into a waiting state and calls unlockf.
  If unlockf returns false, the goroutine is resumed. Implementation execute scheduling
  of the next goroutine forgetting about existence of current one, until it will
  be brought back by `goready`. Thus, schedule do not waste resources for goroutines
  waiting some external event to continue its execution. This used exactly instead
  of spinning cpu;
* `goparkunlock` puts the current goroutine into a waiting state and unlocks the lock
  by calling `parkunlock_c` over internal `mutex` object. If unlockf returns false,
  the goroutine is resumed. Implemented via;
* `goready / ready` mark gp ready to run. Naturally `unpark`. Places a goroutine
  into the next run slot (via `runqput`) or to the local run queue (size 256) if
  its contended. If the local run queue is full, runnext puts g on the global queue.

Parking used in implementations of io, gc, timers, finalizers, channels, panics, tracer,
semaphore and select.

It effectively used for implementations of sync primitives when the moment of acquisition
or releasing the lock is known in advance (instead of blind spinning) (by some external
event i.e.).

If there was no contention on next run slot on the `p`, `goready` can effectively
bring goroutine back to life omitting long passing through the run queues what
intended to minimize latency.

Race detector
===

`GoPark`'s `unlockf` runs on g0, which has no race context (`g.racectx == 0`).
Under `-race`, any instrumented code there (a plain Go func, `sync`, `sync/atomic`)
calls `__tsan_func_enter(NULL, ...)` and crashes with `SIGSEGV addr=0x10`.
Use `park.ParkUnlock_c`, `park.KeepLock` or your own `//go:norace` function as `unlockf`.

Primitives
==========

Everything below is built from three runtime operations: `gopark`, `goready`
and `runtime.lock`. Two rules from the runtime apply to every one of them:

1. **`GoReady(gp)` is legal only when `gp` is `_Gwaiting`.** Otherwise the
   runtime throws `bad g->status in ready`. So the waker must *know* the
   waiter is parked, not just hope so.
2. **`unlockf` runs on g0 after the waiter is `_Gwaiting`.** `park_m` does
   `casgstatus(gp, _Grunning, _Gwaiting)`, then calls `unlockf(gp, lock)`.
   If `unlockf` returns `false`, the goroutine goes straight back to running
   and is never scheduled away. That window, "already waiting, not yet
   visible to wakers", is where every primitive closes its lost-wakeup race.
   Because it runs on g0, `unlockf` must be `//go:norace` and `//go:nosplit`,
   must not use `sync/atomic` (instrumented under `-race`), and must not touch
   the waiter's stack (which may move).

Also: `mutex.Mutex` is held by the *M*, not the goroutine. Holding it raises
`m.locks`, which disables preemption, and the scheduler throws
`schedule: holding locks` if the holder blocks. Critical sections must not
block. The only legal way to sleep with it held is to park with an `unlockf`
that releases it (`GoParkUnlock`, `waitq.Queue.Wait`).

### waitq: condition variable

`waitq.Queue` is a FIFO of `*g.G` guarded by a `mutex.Mutex` the caller
owns. It is a `sync.Cond` built from `GoParkUnlock` and `GoReady` alone, the
same pattern the runtime uses for its own wait lists.

| method      | requires m held | does |
|-------------|-----------------|------|
| `Wait(&m)`  | yes | appends current g, parks with `GoParkUnlock` (m released on g0 once parked), relocks m |
| `Signal()`  | yes | dequeues the oldest g and `GoReady`s it; reports whether there was one |
| `Broadcast()` | yes | readies all, FIFO; returns the count |
| `Len()`     | yes | number of parked goroutines |

Why no wakeup is lost: the g is enqueued under `m`, and `m` is released only
by `unlockf`, i.e. after the g is `_Gwaiting`. A waker must hold `m` to see
the queue, so any g it finds is already parked and rule 1 holds. The g is
removed before it is readied, so it is readied exactly once.

As with `sync.Cond`, always re-check the condition in a loop after `Wait`.
`GoReady` puts the woken g in the P's `runnext` slot. A burst of `Signal`s
therefore *dequeues* FIFO, but the woken goroutines may *run* in a different
order (the last one readied tends to run first).

### event: one-shot wakeup in any order

`event.Event` is the runtime's `note` (`notesleep`/`notewakeup`) rebuilt on
`gopark` instead of a futex. It fixes `Park`'s hazard, where calling `Ready`
before the goroutine has parked crashes the program. Its whole state is one word:

| state       | meaning |
|-------------|---------|
| `evEmpty`   | not set, nobody parked |
| `evSet`     | set (latched forever, there is no Reset) |
| `evTimeout` | `WaitTimeout`'s timer fired before the waiter managed to park |
| any other   | the `*g.G` of the parked waiter |

- **`Wait`**: if set, return. Otherwise `GoPark(eventPark, e)`. `eventPark` is
  the `unlockf`: on g0, with the waiter already `_Gwaiting`, it does
  `CAS(state, evEmpty → gp)`. If the CAS fails, `Set` (or the timer) got there
  first, so it returns `false` and the waiter resumes without ever sleeping.
  This is the one place in the package where `unlockf` returns `false`.
- **`Set`**: `Swap(state, evSet)`. If the old value was a g, that g was put
  there by `eventPark` *after* it became `_Gwaiting`, so `GoReady` is legal.
  Idempotent; any number of setters.
- **`WaitTimeout(d) bool`**: same as `Wait`, plus `time.AfterFunc(d, timeout)`.
  The timer either takes the g out (`CAS(gp → evEmpty)`, then `GoReady`), or,
  if the waiter has not parked yet, leaves `evTimeout` so the waiter's
  `eventPark` fails and it never parks. After a timeout the state is back to
  `evEmpty`, so the Event can be waited on again or set. Whoever wins the CAS
  owns the wakeup, so `Set` and the timer never both ready the same g.

The CAS inside `eventPark` cannot be `sync/atomic` (instrumented under
`-race`, and g0 has no race context), and `internal/runtime/atomic` is not
linkname-able. So it is a few lines of assembly: `event/cas_{amd64,386,arm64}.s`
(`LOCK CMPXCHG` on x86, an `LDAXR`/`STLXR` loop on arm64).

One waiter at a time: a second concurrent waiter is detected and panics.
`Set` happens-before any `Wait`/`WaitTimeout` that returns `true`.

### IsParked: scheduling state

`g.(*G).Status()` returns `atomicstatus` with the GC's `_Gscan` bit cleared,
and the package exports the state constants (`g.Gwaiting`, `g.Grunnable`, ...).
`park.IsParked(gp)` reports `Status() == g.Gwaiting`, and `(*Park).IsParked`
does the same for the g recorded by `Set`.

It is a snapshot. Acting on it (`GoReady`) is safe only when the caller is the
only party that can wake `gp`. It replaces the "lock the mutex that
`GoParkUnlock` releases" handshake with a poll: `for !park.IsParked(gp) { runtime.Gosched() }`.
`*g.G` pointers never dangle, because the runtime never frees g structs (they are
kept in `allgs` and reused), but a reused g reports its new owner's state.
`_Gwaiting` covers *any* park reason (channels, timers, sync), not only this package.

### Race detector annotations

The race detector cannot see `runtime.lock` or `gopark`/`goready` as
synchronization. `mutex.Mutex` therefore calls `runtime.RaceAcquire` after
`Lock` and `runtime.RaceRelease` before `Unlock` (`internal/race`, no-ops
without `-race`). `GoParkUnlock` publishes the release on the goroutine before
parking, because `ParkUnlock_c` unlocks on g0, which the race detector cannot see.
`event.Event` needs no annotations: its goroutine-side state accesses are `sync/atomic`.

TODO
====

- **Semaphore.** Linkname `sync.runtime_Semacquire` / `sync.runtime_Semrelease`
  (`runtime/sema.go`). That is the real sudog-based, address-keyed semaphore,
  including the `handoff` flag that yields the P straight to the woken waiter.
- **Per-P pinning.** `runtime.procPin` / `procUnpin` (`proc.go`, explicitly
  linkname-exported). Pinning disables preemption and returns the P id, which
  enables per-P sharded counters and caches as in `sync.Pool`.
- **Spin-then-park lock.** `sync.runtime_canSpin` / `sync.runtime_doSpin`
  (procyield) for a short adaptive spin, then fall back to `waitq.Queue`. A model
  of how `sync.Mutex` behaves.
- **`runtime.cheaprand`.** A fast per-M PRNG, e.g. for jittered backoff in the spin lock.
- **Possible bug: `Park.Park(m)` with `KeepLock`.** It parks without releasing `m`.
  If `m` is held, `m.locks > 0` reaches `schedule()`, which should throw
  `schedule: holding locks` (`runtime/proc.go`). Unverified: write a test, then
  either drop `Park.Park` or document that `m` must not be held.

Author
===

Ivan Prisyazhnyy, @john.koepi, 2019

