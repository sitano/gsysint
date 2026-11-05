# gsysint

Golang (as of 1.27.2, amd64/386/arm64) runtime internals that gives you an access to internal scheduling primitives.
(for learning purposes)

Features
========

* `g` and `m` internal structures access (read goroutine id)
* goroutines native parking / unparking
* internal spin lock

Examples
=======

Get goroutine id:

    gsysint.GoID()

    or

    gsysint.GIDFromStackTrace()

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

Author
===

Ivan Prisyazhnyy, @john.koepi, 2019

