package event

import (
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sitano/gsysint/park"
	"github.com/sitano/gsysint/runtime/g"
)

func TestSetBeforeWait(t *testing.T) {
	var e Event
	e.Set()
	e.Set() // idempotent
	e.Wait()
	if !e.WaitTimeout(time.Hour) {
		t.Fatal("WaitTimeout on set event returned false")
	}
}

func TestWaitThenSet(t *testing.T) {
	var e Event
	var gp atomic.Pointer[g.G]
	data := 0
	done := make(chan struct{})
	go func() {
		gp.Store(g.CurG())
		e.Wait()
		if data != 42 { // Set happens-before Wait returning
			t.Error("data not visible after Wait")
		}
		close(done)
	}()
	for gp.Load() == nil {
		runtime.Gosched()
	}
	for !park.IsParked(gp.Load()) {
		runtime.Gosched()
	}
	data = 42
	e.Set()
	<-done
}

func TestTimeout(t *testing.T) {
	var e Event
	start := time.Now()
	if e.WaitTimeout(10 * time.Millisecond) {
		t.Fatal("WaitTimeout returned true on unset event")
	}
	if time.Since(start) < 10*time.Millisecond {
		t.Fatal("WaitTimeout returned early")
	}
	// Zero timeout: timer may fire before the park, exercising evTimeout.
	for range 100 {
		if e.WaitTimeout(0) {
			t.Fatal("WaitTimeout(0) returned true on unset event")
		}
	}
	e.Set()
	if !e.WaitTimeout(time.Hour) {
		t.Fatal("event not set after timeouts")
	}
}

func TestSetDuringTimedWait(t *testing.T) {
	var e Event
	go func() {
		time.Sleep(time.Millisecond)
		e.Set()
	}()
	if !e.WaitTimeout(time.Hour) {
		t.Fatal("WaitTimeout missed Set")
	}
}

// Race Set against Wait/WaitTimeout with no handshake, many times.
func TestStress(t *testing.T) {
	for i := range 2000 {
		var e Event
		go e.Set()
		if i%2 == 0 {
			e.Wait()
			continue
		}
		// Yield between retries: on one P the timer goroutines of back-to-back
		// microsecond waits would otherwise keep the Set goroutine off the CPU.
		for !e.WaitTimeout(time.Duration(i%3) * time.Microsecond) {
			runtime.Gosched()
		}
	}
}
