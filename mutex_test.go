package gsysint

import "testing"

func TestMutex(t *testing.T) {
	l := &Mutex{}
	Lock(l)
	Unlock(l)
}

func BenchmarkMutexUncontended(b *testing.B) {
	l := &Mutex{}
	for i := 0; i < b.N; i ++ {
		Lock(l)
		Unlock(l)
	}
}