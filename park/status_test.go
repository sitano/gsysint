package park

import (
	"runtime"
	"testing"

	"github.com/sitano/gsysint/mutex"
	"github.com/sitano/gsysint/runtime/g"
)

func TestIsParked(t *testing.T) {
	if IsParked(g.CurG()) {
		t.Fatal("running goroutine reported as parked")
	}
	var p Park
	var m mutex.Mutex
	done := make(chan struct{})
	set := make(chan struct{})
	go func() {
		p.Set()
		close(set)
		m.Lock()
		p.ParkUnlock(&m)
		close(done)
	}()
	<-set
	for !p.IsParked() {
		runtime.Gosched()
	}
	p.Ready()
	<-done
}
