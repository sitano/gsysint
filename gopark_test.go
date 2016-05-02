package gsysint

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"
	"runtime/pprof"
	"bytes"
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

	Lock(l)
	GoReady((*G)(gp), 1)
	Unlock(l)

	w.Wait()
}

func TestParkLock(t *testing.T) {
	var gp unsafe.Pointer

	go func() {
		atomic.StorePointer(&gp, GetG())
		GoPark(func(g *G, p unsafe.Pointer) bool {
			return true
		}, nil, "go (block)", TraceEvGoBlock, 1)
	}()

	runtime.Gosched()

	stack := &bytes.Buffer{}
	pprof.Lookup("goroutine").WriteTo(stack, 1)
	t.Log(stack.String())

	GoReady((*G)(gp), 1)
}
