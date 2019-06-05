package gsysint

import (
	"testing"

	"github.com/sitano/gsysint/g"
)

func TestMutex(t *testing.T) {
	l := &g.Mutex{}
	Lock(l)
	Unlock(l)
}

func BenchmarkMutexUncontended(b *testing.B) {
	l := &g.Mutex{}
	for i := 0; i < b.N; i++ {
		Lock(l)
		Unlock(l)
	}
}
