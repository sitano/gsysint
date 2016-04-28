# gsysint
Go system internals

Features
========

* `g` and `m` internal structures access
* goroutines native parking / unparking
* internal locks

Example
=======

```golang

var gp unsafe.Pointer

l := &Mutex{}
go func() {
    Lock(l)
    atomic.StorePointer(&gp, GetG())
    GoParkUnlock(l, "go (block)", TraceEvGoBlock, 1)
}()

runtime.Gosched()

Lock(l)
GoReady((*G)(gp), 1)
Unlock(l)

```