# gsysint
Go (as of 1.6) system internals

Features
========

* `g` and `m` internal structures access
* goroutines native parking / unparking
* internal locks

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

Example
=======

Demonstration of goroutine parking with mutex unlocking.

```golang

var gp unsafe.Pointer

l := &Mutex{}
go func() {
    Lock(l)
    atomic.StorePointer(&gp, GetG())
    GoParkUnlock(l, "go (block)", TraceEvGoBlock, 1)
}()

runtime.Gosched()

pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)

Lock(l)
GoReady((*G)(gp), 1)
Unlock(l)

```
