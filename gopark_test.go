package gsysint

import (
	"bytes"
	"runtime"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"

	"github.com/sitano/gsysint/g"
	"github.com/sitano/gsysint/trace"
)

func TestPark(t *testing.T) {
	t.Run("raw api park with unlock", func(t *testing.T) {
		var gp unsafe.Pointer

		w := sync.WaitGroup{}
		w.Add(1)

		l := &g.Mutex{}
		go func() {
			atomic.StorePointer(&gp, g.GetG())
			Lock(l)
			// park
			GoParkUnlock(l, g.WaitReasonZero, trace.TraceEvNone, 1) // actual park
			w.Done()
		}()

		runtime.Gosched()

		if gp == nil {
			t.Fatalf("GetG() returned nil pointer to the g structure")
		}

		Lock(l)
		// unpark goroutine and mark as ready
		GoReady((*g.G)(gp), 1)
		Unlock(l)

		w.Wait()
	})

	t.Run("simple api park with unlock", func(t *testing.T) {
		var p Park
		var m g.Mutex
		var w sync.WaitGroup

		w.Add(1)
		go func() {
			p.Set()
			Lock(&m)
			// park
			p.ParkUnlock(&m)
			w.Done()
		}()

		runtime.Gosched()

		Lock(&m)
		// unpark goroutine and mark as ready
		p.Ready()
		Unlock(&m)

		w.Wait()
	})
}

func TestParkLock(t *testing.T) {
	var gp unsafe.Pointer

	go func() {
		atomic.StorePointer(&gp, g.GetG())
		GoPark(func(g *g.G, p unsafe.Pointer) bool {
			return true
		}, nil, g.WaitReasonZero, trace.TraceEvNone, 1)
	}()

	runtime.Gosched()

	stack := &bytes.Buffer{}
	_ = pprof.Lookup("goroutine").WriteTo(stack, 1)
	t.Log(stack.String())

	GoReady((*g.G)(gp), 1)
}
