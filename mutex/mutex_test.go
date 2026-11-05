package mutex

import "testing"

func TestMutex(t *testing.T) {
	var l Mutex
	l.Lock()
	l.Unlock()
}

func BenchmarkMutexUncontended(b *testing.B) {
	var l Mutex
	for b.Loop() {
		l.Lock()
		l.Unlock()
	}
}
