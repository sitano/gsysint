package gsysint

import (
	"testing"

	"github.com/sitano/gsysint/g"
)

func TestGIDFromStack(t *testing.T) {
	id, err := GIDFromStackTrace()
	if err != nil {
		t.Fatal(err)
	}
	id2 := g.CurG().GoID
	if id != id {
		t.Error("ids are different", id, id2)
	}
}
