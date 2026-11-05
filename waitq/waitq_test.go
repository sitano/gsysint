package waitq

import (
	"runtime"
	"sync"
	"testing"

	"github.com/sitano/gsysint/mutex"
)

func TestQueueSignalFIFO(t *testing.T) {
	const n = 8
	var m mutex.Mutex
	var q Queue
	var order []int
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Lock()
			q.Wait(&m)
			order = append(order, i)
			m.Unlock()
		}()
		// Queue goroutines one at a time so the expected order is known.
		for {
			m.Lock()
			l := q.Len()
			m.Unlock()
			if l == i+1 {
				break
			}
			runtime.Gosched()
		}
	}
	// Signal one at a time and wait for that waiter to run: GoReady puts each
	// g in runnext, so readying them all at once would run them LIFO.
	for i := range n {
		m.Lock()
		if !q.Signal() {
			t.Fatal("Signal found no waiter")
		}
		m.Unlock()
		for {
			m.Lock()
			l := len(order)
			m.Unlock()
			if l == i+1 {
				break
			}
			runtime.Gosched()
		}
	}
	wg.Wait()
	m.Lock()
	defer m.Unlock()
	if q.Signal() {
		t.Fatal("Signal on empty queue returned true")
	}
	for i, v := range order {
		if v != i {
			t.Fatalf("wake order %v, want FIFO", order)
		}
	}
}

func TestQueueBroadcast(t *testing.T) {
	const n = 16
	var m mutex.Mutex
	var q Queue
	var wg sync.WaitGroup
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Lock()
			q.Wait(&m)
			m.Unlock()
		}()
	}
	for {
		m.Lock()
		if q.Len() == n {
			break
		}
		m.Unlock()
		runtime.Gosched()
	}
	got := q.Broadcast()
	m.Unlock()
	if got != n {
		t.Fatalf("Broadcast woke %d, want %d", got, n)
	}
	wg.Wait()
}

// Producer/consumer through a condition loop; plain data guarded by m (checks -race annotations).
func TestQueueCond(t *testing.T) {
	const items = 1000
	var m mutex.Mutex
	var notEmpty Queue
	var buf []int
	sum := 0
	done := make(chan struct{})
	go func() {
		for range items {
			m.Lock()
			for len(buf) == 0 {
				notEmpty.Wait(&m)
			}
			sum += buf[0]
			buf = buf[1:]
			m.Unlock()
		}
		close(done)
	}()
	for i := range items {
		m.Lock()
		buf = append(buf, i)
		notEmpty.Signal()
		m.Unlock()
	}
	<-done
	if want := items * (items - 1) / 2; sum != want {
		t.Fatalf("sum %d, want %d", sum, want)
	}
}
