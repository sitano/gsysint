package gsysint

import (
	"bytes"
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
			w.Done()

			// park
			GoParkUnlock(l, g.WaitReasonZero, trace.BlockGeneric, 1) // actual park
		}()

		// Ensure goroutine has started and acquired the lock before we attempt to unpark it.
		w.Wait()

		if atomic.LoadPointer(&gp) == nil {
			t.Fatalf("GetG() returned nil pointer to the g structure")
		}

		// We will acquire the lock as soon as the goroutine is parked, so that we can safely unpark it.
		Lock(l)
		// Unpark the goroutine and mark as ready.
		GoReady((*g.G)(atomic.LoadPointer(&gp)), 1)
		Unlock(l)
	})

	t.Run("simple api park with unlock", func(t *testing.T) {
		var p Park
		var m g.Mutex
		var w sync.WaitGroup

		w.Add(1)
		go func() {
			p.Set()

			Lock(&m)
			w.Done()

			// park
			p.ParkUnlock(&m)
		}()

		// Ensure the goroutine has started and acquired the lock before we attempt to unpark it.
		w.Wait()

		// We will acquire the lock as soon as the goroutine is parked, so that we can safely unpark it.
		Lock(&m)
		// Unpark the goroutine and mark as ready.
		p.Ready()
		Unlock(&m)
	})
}

func TestParkLock(t *testing.T) {
	var gp unsafe.Pointer
	var wg1, wg2 sync.WaitGroup

	wg1.Add(1)
	wg2.Add(1)
	go func() {
		atomic.StorePointer(&gp, g.GetG())
		GoPark(func(g *g.G, p unsafe.Pointer) bool {
			// Test-only hack: unlockf runs on g0 after gp is _Gwaiting, so this is a
			// reliable "parked" signal here. Don't do this in real code: g0 can't grow
			// its stack, be preempted or yield, so unlockf must stay tiny and nosplit-safe
			// (e.g. an atomic store or runtime unlock), and wg.Done pulls in sync/sema
			// and race-detector code that is not written to run on g0.
			wg1.Done()
			return true
		}, nil, g.WaitReasonZero, trace.BlockGeneric, 1)
		wg2.Done()
	}()

	wg1.Wait() // Wait until the goroutine has reached the park point.

	stack := &bytes.Buffer{}
	_ = pprof.Lookup("goroutine").WriteTo(stack, 1)
	t.Log(stack.String())

	GoReady((*g.G)(atomic.LoadPointer(&gp)), 1)

	wg2.Wait() // Wait until the goroutine has finished after being unparked.
}
