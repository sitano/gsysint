package gsysint

import (
	"testing"
	"sync"
	"runtime"
	"sync/atomic"
	"unsafe"
)

func TestPark(t *testing.T) {
	var gp unsafe.Pointer

	w := sync.WaitGroup{}
	w.Add(1)

	l := &Mutex{}
	go func() {
		atomic.StorePointer(&gp, GetG())
		Lock(l)
		GoParkUnlock(l, "go (block)", TraceEvGoBlock, 1)
		w.Done()
	}()

	runtime.Gosched()

	if gp == nil {
		t.Fatalf("GetG() returned nil pointer to the g structure")
	}

	GoReady((*G)(gp), 1)

	w.Wait()
}

func TestParkLock(t *testing.T) {
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
}
