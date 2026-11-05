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
	var l g.Mutex

	wg1.Add(1)
	wg2.Add(1)
	go func() {
		atomic.StorePointer(&gp, g.GetG())
		Lock(&l)
		wg1.Done()
		// unlockf runs on g0, which has no race context under -race, so it must be
		// norace code: ParkUnlock_c releases l only once gp is _Gwaiting.
		GoPark(ParkUnlock_c, unsafe.Pointer(&l), g.WaitReasonZero, trace.BlockGeneric, 1)
		wg2.Done()
	}()

	wg1.Wait()
	Lock(&l) // Acquired only after the goroutine is parked.
	Unlock(&l)

	stack := &bytes.Buffer{}
	_ = pprof.Lookup("goroutine").WriteTo(stack, 1)
	t.Log(stack.String())

	GoReady((*g.G)(atomic.LoadPointer(&gp)), 1)

	wg2.Wait() // Wait until the goroutine has finished after being unparked.
}
